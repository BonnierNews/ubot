package bot

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"plugin"
	"regexp"
	"strings"

	"github.com/BonnierNews/ubot/pkg/api"
	slacktemplates "github.com/BonnierNews/ubot/pkg/slack"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	reCmd           = regexp.MustCompile(`\S+`)
	verbose         bool
	alertmanagerURL *url.URL
	log             = logrus.New()
	app             = kingpin.New("ubot", "A Microbot for slack").DefaultEnvars()
)

// UbotPlugin is the interface for plugins
type UbotPlugin interface {
	RunPlugin()
}

// Ubot type
type Ubot struct {
	ctx        context.Context
	pluginsDir string
	commands   map[string]api.UbotCommand
}

// New creates a ubot
func New() *Ubot {
	return &Ubot{
		commands: make(map[string]api.UbotCommand),
	}
}

// Init Ubot
func (bot *Ubot) Init(ctx context.Context) error {
	bot.ctx = ctx
	return bot.loadCommands()
}

func (bot *Ubot) loadCommands() error {
	if _, err := os.Stat(bot.pluginsDir); err != nil {
		return err
	}

	plugins, err := listFiles(bot.pluginsDir, `.*.so`)
	if err != nil {
		return err
	}

	for _, botPlugin := range plugins {
		plug, err := plugin.Open(path.Join(bot.pluginsDir, botPlugin.Name()))
		if err != nil {
			log.Fatalf("failed to open plugin %s: %v\n", botPlugin.Name(), err)
			continue
		}
		cmdSymbol, err := plug.Lookup(api.SymbolName)
		if err != nil {
			log.Errorf("plugin %s does not export symbol \"%s\"",
				botPlugin.Name(), api.SymbolName)
			continue
		}
		commands, ok := cmdSymbol.(api.UbotCommands)
		if !ok {
			log.Errorf("Symbol %s (from %s) does not implement Commands interface\n",
				api.SymbolName, botPlugin.Name())
			continue
		}
		if err := commands.Init(bot.ctx); err != nil {
			log.Errorf("%s initialization failed: %v\n", botPlugin.Name(), err)
			continue
		}
		for name, cmd := range commands.Registry() {
			bot.commands[name] = cmd
		}
		bot.ctx = context.WithValue(bot.ctx, "bot.commands", bot.commands)
	}
	return nil
}

func (bot *Ubot) help(line string) (string, error) {
	const tplHelpBase = `Available commands:
		{{range .}}{{.|print| code}}: {{.ShortDesc}}
		{{end}}
Use {{"help <command>"| code}} for help on each command`

	const tplHelpCommand = `Command: {{.Name|print|code}}
		{{.Usage}}, {{.LongDesc}}`

	var b bytes.Buffer
	if line == "help" {
		tpl := template.Must(template.New("help").Funcs(slacktemplates.FuncMap).Parse(tplHelpBase))
		tpl.ExecuteTemplate(&b, "help", bot.commands)
		return b.String(), nil
	} else {
		line = strings.TrimPrefix(line, "help")
		line = strings.TrimSpace(line)
		cmd, ok := bot.commands[line]
		if !ok {
			log.Errorf("command not found: %s", line)
			return "Command not found", nil
		}
		tpl := template.Must(template.New("help").Funcs(slacktemplates.FuncMap).Parse(tplHelpCommand))
		tpl.ExecuteTemplate(&b, "help", bot.commands[cmd.Name()])
		return b.String(), nil
	}
}

func (bot *Ubot) handle(ctx context.Context, ev *slack.MessageEvent, info *slack.Info) ([]api.UbotReturn, error) {
	var ret []api.UbotReturn
	prefix := fmt.Sprintf("<@%s> ", info.User.ID)
	line := strings.TrimPrefix(ev.Text, prefix)
	line = strings.TrimSpace(line)
	line = strings.ToLower(line)
	if line == "" || strings.HasPrefix(line, "help") {
		help, _ := bot.help(line)
		ret = append(ret, api.UbotReturn{
			Message:           help,
			MessageParameters: slack.PostMessageParameters{},
		})
		return ret, nil
	}
	args := reCmd.FindAllString(line, -1)
	if args != nil {
		cmdName := args[0]
		cmd, ok := bot.commands[cmdName]
		if !ok {
			log.Errorf("help command not found: %s", cmdName)
			ret = append(ret, api.UbotReturn{
				Message:           "",
				MessageParameters: slack.PostMessageParameters{},
			})
			return ret, fmt.Errorf("Command not found: %s", cmdName)
		}
		log.Infof("Handled command, using: %s", cmd)
		return cmd.Exec(ctx, ev, info)
	}
	ret = append(ret, api.UbotReturn{
		Message:           "",
		MessageParameters: slack.PostMessageParameters{},
	})
	return ret, fmt.Errorf("Unable to parse command line: %s", line)
}

func listFiles(dir, pattern string) ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	filteredFiles := []os.FileInfo{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		matched, err := regexp.MatchString(pattern, file.Name())
		if err != nil {
			return nil, err
		}
		if matched {
			filteredFiles = append(filteredFiles, file)
		}
	}
	return filteredFiles, nil
}

// Execute runs the bot
func Execute() {
	var (
		app = kingpin.New("ubot", "A Microbot for slack").DefaultEnvars()
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.Version("0.0.1")
	app.UsageTemplate(kingpin.CompactUsageTemplate)
	slackToken := app.Flag("slack.token", "Slack token").
		Required().
		OverrideDefaultFromEnvar("SLACK_TOKEN").
		String()
	pluginDir := app.Flag("plugin.dir", "Realtive path to the plugins directory").
		OverrideDefaultFromEnvar("PLUGIN_DIR").
		Default("./plugins").
		String()
	app.HelpFlag.Short('h')
	_, err := app.Parse(os.Args[1:])
	if err != nil {
		kingpin.Fatalf("%v\n", err)
	}
	kingpin.MustParse(app.Parse(os.Args[1:]))
	log.Info("Starting ubot")
	bot := New()
	bot.pluginsDir = *pluginDir
	bot.Init(ctx)
	api := slack.New(*slackToken)
	rtm := api.NewRTM()
	rtm.GetBotInfo("ubot")
	go rtm.ManageConnection()
Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:

			case *slack.MessageEvent:
				log.Debugf("Message %+v", ev)
				info := rtm.GetInfo()
				// Only hande @bot <command>
				prefix := fmt.Sprintf("<@%s> ", info.User.ID)
				if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
					info := rtm.GetInfo()
					ret, err := bot.handle(ctx, ev, info)
					if err != nil {
						log.Errorf("%v", err)
					}
					for _, msgReturn := range ret {
						rtm.SendMessage(rtm.NewOutgoingMessage(msgReturn.Message, ev.Channel))
					}
				}

			case *slack.RTMError:
				log.Fatalf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				log.Fatalf("Invalid credentials")
				break Loop

			default:
				//Take no action
			}
		}
	}
}
