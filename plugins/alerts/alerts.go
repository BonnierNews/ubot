package main

import (
	"context"
	"fmt"

	"github.com/BonnierNews/ubot/pkg/api"
	"github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
	"github.com/prometheus/alertmanager/client"
	promapi "github.com/prometheus/client_golang/api"
)

var (
	log             = logrus.New()
	alertmanagerURL = "http://localhost:9093"
)

type alertQueryCmd struct {
	expired, silenced bool
	matcherGroups     []string
}

const alertHelp = `View and search through current alerts.
*alerts* has a simplified prometheus query syntax, but contains robust support for
bash variable expansions. The non-option section of arguments constructs a list
of "Matcher Groups" that will be used to filter your query. The following
examples will attempt to show this behaviour in action:
*alerts* query alertname=foo node=bar
	This query will match all alerts with the alertname=foo and node=bar label
	value pairs set.
*alerts* query foo node=bar
	If alertname is omitted and the first argument does not contain a '=' or a
	'=~' then it will be assumed to be the value of the alertname pair.
*alerts* query 'alertname=~foo.*'
	As well as direct equality, regex matching is also supported. The '=~' syntax
	(similar to prometheus) is used to represent a regex match. Regex matching
	can be used in combination with a direct match.
`

func (a *alertQueryCmd) queryAlerts(query string) ([]*client.ExtendedAlert, error) {
	var filterString = ""
	/* 	if len(a.matcherGroups) == 1 {
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
	   	} */

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
	log.Errorf("Got alerts: %+v", fetchedAlerts)
	/*formatter, found := format.Formatters["json"]
	if !found {
		return errors.New("unknown output formatter")
	} */
	return fetchedAlerts, err
}

type alertCmd string

func (t alertCmd) Name() string      { return string(t) }
func (t alertCmd) Usage() string     { return fmt.Sprintf("Usage: %s <text>", t.Name()) }
func (t alertCmd) ShortDesc() string { return `prints leet of <text>` }
func (t alertCmd) LongDesc() string  { return alertHelp }
func (t alertCmd) Exec(ctx context.Context, ev *slack.MessageEvent, info *slack.Info) (string, slack.PostMessageParameters, error) {
	args, _ := api.GetArgs(ev.Text, t.Name(), info.User.ID)
	a := &alertQueryCmd{expired: false, silenced: false}
	_, _ = a.queryAlerts(args)
	return "alerrrrrrt", slack.PostMessageParameters{}, nil
}

type alertCmds struct{}

func (t *alertCmds) Init(ctx context.Context) error {

	log.Info("Loaded alerts plugin")
	return nil
}

func (t *alertCmds) Registry() map[string]api.UbotCommand {
	return map[string]api.UbotCommand{
		"alerts": alertCmd("alerts"),
	}
}

// UbotCommands ...
var UbotCommands alertCmds
