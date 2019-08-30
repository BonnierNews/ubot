package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/prometheus/alertmanager/api/v2/client/alert"

	"github.com/BonnierNews/ubot/pkg/api"
	"github.com/nlopes/slack"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/models"

	"github.com/prometheus/alertmanager/cli"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

const pluginName = "alerts"

var (
	log                = logrus.New()
	alertmanagerURL    *url.URL
	app                = kingpin.New("alerts", "Manage alerts")
	silenceCmd         = app.Command("silence", "Silence alerts")
	silenceCmdAlertID  = silenceCmd.Arg("alert", "AlertID to silence").Required().String()
	silenceCmdDuration = silenceCmd.Flag("duration", "Duration of silence").Duration()
	silenceCmdComment  = silenceCmd.Flag("comment", "Comment to add").String()
	listCmd            = app.Command("list", "List alerts")
)

type alertManager struct {
	alias  string
	url    url.URL
	client client.Alertmanager
}

type alertsConfig struct {
	alertManagers []alertManager
}

type alertQuery struct {
	inhibited, silenced, active, unprocessed bool
	receiver                                 string
	matcherGroups                            []string
}

type alertQueryRes struct {
	alerts models.GettableAlerts
	alias  string
}

type silenceAdd struct {
	author         string
	requireComment bool
	duration       string
	start          string
	end            string
	comment        string
	matchers       []string
}

const alertHelp = `*list*, *get* and *silence* current alerts.
Actions:
- *list:*
	*alerts* list
		List all active alerts
- *get:*
	*alerts* get <alertID>
		This will display a detailed view of the alert with <alertid>, including
		all labels and annotations.

- *silence:*
	*alerts* silence <alertID> (duration=<duration>, 1h) (comment=<comment>)
		This silence will silence the alert with <alertId> and optionally set duration and comment.
`

const alertsListTemplate = `*{{.Alert.Labels["alertname"]}}*
*Labels:*
		{{range .Alerts.Labels}}{{.|print| code}}: {{.ShortDesc}}
		{{end}}
`

func (c *silenceAdd) silenceAlert(alertID string, duration string, comment string) error {
	return fmt.Errorf("Error")

}

func (a *alertQuery) list(cfg *alertsConfig, query string) ([]alertQueryRes, error) {
	res := []alertQueryRes{}
	alertParams := alert.NewGetAlertsParams().
		WithActive(&a.active).
		WithInhibited(&a.inhibited).
		WithSilenced(&a.silenced).
		WithReceiver(&a.receiver).
		WithUnprocessed(&a.unprocessed).
		WithFilter(a.matcherGroups)
	for _, amClient := range cfg.alertManagers {
		getOk, err := amClient.client.Alert.GetAlerts(alertParams)
		if err != nil {
			return nil, err
		}
		log.Infof("%s", amClient.alias)
		res = append(res, alertQueryRes{alias: amClient.alias, alerts: getOk.Payload})
	}
	log.Infof("Res: %#v", res)
	return res, nil
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

	// Don't terminate on errors
	app.Terminate(nil)

	// Redirect io.Writer to a buffer
	buf := new(bytes.Buffer)
	app.UsageWriter(buf)
	app.ErrorWriter(buf)

	// Get args string
	arg, _ := api.GetArgs(ev.Text, t.Name(), info.User.ID)
	log.Infof("Args: %v\n", arg)
	args := strings.Split(arg, " ")
	cmd, err := app.Parse(args)
	if err != nil {
		app.FatalUsage(err.Error())
	}
	if len(buf.String()) > 0 {
		ret := append(ret, api.UbotReturn{
			Message:           fmt.Sprintf("%s", buf.String()),
			MessageParameters: slack.PostMessageParameters{},
		})
		return ret, nil
	}

	// main switch
	c, err := newAlertsConfig()
	if err != nil {
		return ret, err
	}

	switch cmd {
	case silenceCmd.FullCommand():
		log.Infof("SilenceCommand: %s", app.Name)

	case listCmd.FullCommand():
		var (
			a = &alertQuery{}
		)
		a = &alertQuery{silenced: false, active: true}
		res, err := a.list(c, arg)
		if err != nil {
			log.Errorf("Error getting alerts: %+v", err)
			ret := append(ret, api.UbotReturn{
				Message:           fmt.Sprintf("Error getting alerts: %s", err),
				MessageParameters: slack.PostMessageParameters{},
			})
			return ret, nil
		}
		for _, alerts := range res {
			for _, a := range alerts.alerts {
				//slack.Attachment{
				//	AuthorName: fmt.Sprintf("Alertmanager: %s", alerts.alias),
				//	Title:      fmt.Sprintf("%s", a.Labels["alertname"]),
				//	Text:       fmt.Sprint("%s", a.Annotations["description"]),
				//}
				ret = append(ret, api.UbotReturn{
					Message: fmt.Sprintf("%s - %s\nDescription: %s", *a.Fingerprint, a.Labels["alertname"], a.Annotations["description"]),
				})
			}

		}
	default:
		ret = append(ret, api.UbotReturn{
			Message:           "Not implemented",
			MessageParameters: slack.PostMessageParameters{},
		})
		return ret, nil
	}
	return ret, nil
}

type alertCmds struct{}

func (t *alertCmds) Init(ctx context.Context) error {
	log.Infof("Trying to start the %s plugin...", pluginName)
	log.Infoln("This plugin needs a ENV var(ALERTMANAGER_URL) set in the form <alias>:proto://address:port,...\nExample: am1:https://am1:9093,am2:https://am2:9093")
	err := verifyEnv()
	if err != nil {
		return err
	}
	log.Infof("Starting %s plugin", pluginName)
	return nil
}

func (t *alertCmds) Registry() map[string]api.UbotCommand {
	return map[string]api.UbotCommand{
		pluginName: alertCmd(pluginName),
	}
}

func newAlertsConfig() (*alertsConfig, error) {
	amEnv, _ := api.GetEnv("ALERTMANAGER_URL", "")
	amHosts := strings.Split(amEnv, ",")
	c := alertsConfig{}
	r := regexp.MustCompile(`^(?P<alias>[a-zAz0-9\-]+):(?P<url>(http|https)://([a-z0-9\.]+):([0-9]{4,5}))`)
	am := alertManager{}
	for _, amHost := range amHosts {
		log.Infof("Looping host: %s", amHost)
		match := r.FindStringSubmatch(amHost)
		for i, name := range r.SubexpNames() {
			if i != 0 && name != "" {
				if name == "alias" {
					am.alias = match[i]
				}
				if name == "url" {
					u, err := url.Parse(match[i])
					if err != nil {
						return &c, err
					}
					am.url = *u
					am.client = *cli.NewAlertmanagerClient(u)
				}
			}
		}
		c.alertManagers = append(c.alertManagers, am)
	}
	return &c, nil
}

func verifyEnv() error {
	amEnv, err := api.GetEnv("ALERTMANAGER_URL", "")
	if err != nil {
		return err
	}
	amHosts := strings.Split(amEnv, ",")
	for _, amHost := range amHosts {
		matched, _ := regexp.MatchString(`^([a-zAz0-9\-]+):(http|https)://([a-z0-9\.]+):([0-9]{4,5})`, amHost)
		if !matched {
			return errors.New("ALERTMANAGER_URL does not contain valid config")
		}
	}
	return nil
}

// UbotCommands ...
var UbotCommands alertCmds
