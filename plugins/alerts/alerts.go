package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/BonnierNews/ubot/pkg/api"
	"github.com/nlopes/slack"
	"github.com/prometheus/alertmanager/client"
	"github.com/prometheus/alertmanager/pkg/parse"
	promapi "github.com/prometheus/client_golang/api"

	"github.com/sirupsen/logrus"
)

var (
	log             = logrus.New()
	alertmanagerURL = "http://docker.for.mac.localhost:9093"
)

type alertQueryCmd struct {
	expired, silenced bool
	matcherGroups     []string
}

const alertHelp = `*list*, *get* and *silence* current alerts.
*alerts* has a simplified prometheus query syntax, but contains robust support for
bash variable expansions. The non-option section of arguments constructs a list
of Matcher Groups that will be used to filter your query. The following
examples will attempt to show this behaviour in action:
- *list:*
	*alerts* list alertname=foo node=bar
		This query will match all alerts with the alertname=foo and node=bar label
		value pairs set.
	*alerts* list foo node=bar
		If alertname is omitted and the first argument does not contain a ~=~ or a
		~=~~ then it will be assumed to be the value of the alertname pair.
	*alerts* list ~alertname=~foo.*~
		As well as direct equality, regex matching is also supported. The '=~' syntax
		(similar to prometheus) is used to represent a regex match. Regex matching
		can be used in combination with a direct match.

- *get:*
	*alerts* get <alertID>
		This will display a detailed view of the alert with <alertid>, including
		all labels and annotations.

- *silence:*
	*alerts* silence alertname=foo node=bar
		This silence will match all alerts with the alertname=foo and node=bar label
		value pairs set.
	*alerts* silence foo node=bar
		If alertname is omitted and the first argument does not contain a '=' or a
		'=~' then it will be assumed to be the value of the alertname pair.
	*alerts* silence 'alertname=~foo.*'
		As well as direct equality, regex matching is also supported. The '=~' syntax
		(similar to prometheus) is used to represent a regex match. Regex matching
		can be used in combination with a direct match.
`

const alertsListTemplate = `*{{.Alert.Labels["alertname"]}}*
*Labels:*
		{{range .Alerts.Labels}}{{.|print| code}}: {{.ShortDesc}}
		{{end}}
`

func (a *alertQueryCmd) listAlerts(query string) ([]*client.ExtendedAlert, error) {
	var filterString = ""
	if len(a.matcherGroups) == 1 {
		// If the parser fails then we likely don't have a (=|=~|!=|!~) so lets
		// assume that the user wants alertname=<arg> and prepend `alertname=`
		// to the front.
		_, err := parse.Matcher(a.matcherGroups[0])
		if err != nil {
			filterString = fmt.Sprintf("{alertname=%s}", a.matcherGroups[0])
		} else {
			filterString = fmt.Sprintf("{%s}", strings.Join(a.matcherGroups, ","))
		}
	} else if len(a.matcherGroups) > 1 {
		filterString = fmt.Sprintf("{%s}", strings.Join(a.matcherGroups, ","))
	}

	c, err := promapi.NewClient(promapi.Config{Address: alertmanagerURL})
	if err != nil {
		return nil, err
	}
	log.Error("creating alertmanager client")
	alertAPI := client.NewAlertAPI(c)
	log.Error("Calling alertmanager")
	fetchedAlerts, err := alertAPI.List(context.Background(), filterString, a.silenced, a.expired)
	if err != nil {
		log.Errorf("Error getting alerts: %+v", err)
		return nil, err
	}
	return fetchedAlerts, err
}

// Plugin
type alertCmd string

func (t alertCmd) Name() string { return string(t) }
func (t alertCmd) Usage() string {
	return fmt.Sprintf("Usage: %s <list|get|silence> <filter>", t.Name())
}
func (t alertCmd) ShortDesc() string { return `Manage alerts in Alertmanager` }
func (t alertCmd) LongDesc() string  { return alertHelp }
func (t alertCmd) Exec(ctx context.Context, ev *slack.MessageEvent, info *slack.Info) ([]api.UbotReturn, error) {
	var ret []api.UbotReturn
	args, _ := api.GetArgs(ev.Text, t.Name(), info.User.ID)
	switch strings.ToLower(strings.Fields(args)[0]) {
	case "list":
		a := &alertQueryCmd{expired: false, silenced: false}
		alerts, err := a.listAlerts(args)
		if err != nil {
			log.Errorf("Error getting alerts: %+v", err)
			ret := append(ret, api.UbotReturn{
				Message:           "Error getting alerts",
				MessageParameters: slack.PostMessageParameters{},
			})
			return ret, nil
		}
		for _, alert := range alerts {
			log.Infof("alerts: %+v\n", alert)
			ret = append(ret, api.UbotReturn{
				Message: fmt.Sprintf("*%s*\n *Description:* %s\n *Labels:* %#v",
					alert.Alert.Labels["alertname"],
					alert.Annotations["description"],
					alert.Alert.Labels),
				MessageParameters: slack.PostMessageParameters{},
			})
		}
	case "get":
		ret = append(ret, api.UbotReturn{
			Message:           "Not implemented",
			MessageParameters: slack.PostMessageParameters{},
		})
		return ret, nil
	case "silence":
		ret = append(ret, api.UbotReturn{
			Message:           "Not implemented",
			MessageParameters: slack.PostMessageParameters{},
		})
		return ret, nil
	default:
		ret = append(ret, api.UbotReturn{
			Message:           "Not implemented",
			MessageParameters: slack.PostMessageParameters{},
		})
		return ret, nil

	}
	ret = append(ret, api.UbotReturn{
		Message:           "Unkown action",
		MessageParameters: slack.PostMessageParameters{},
	})
	return ret, nil
}

type alertCmds struct{}

func (t *alertCmds) Init(ctx context.Context) error {
	_, err := api.GetEnv("ALERTMANAGER_URL", "")
	if err != nil {
		return err
	}
	log.Info("Loading alerts plugin")
	return nil
}

func (t *alertCmds) Registry() map[string]api.UbotCommand {
	return map[string]api.UbotCommand{
		"alerts": alertCmd("alerts"),
	}
}

// UbotCommands ...
var UbotCommands alertCmds
