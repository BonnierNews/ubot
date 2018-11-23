package main

import (
	"context"
	"fmt"

	"github.com/BonnierNews/ubot/pkg/api"
	"github.com/briandowns/formatifier"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
)

type leetCmd string

func (t leetCmd) Name() string      { return string(t) }
func (t leetCmd) Usage() string     { return fmt.Sprintf("Usage: %s <text>", t.Name()) }
func (t leetCmd) ShortDesc() string { return `prints leet of <text>` }
func (t leetCmd) LongDesc() string  { return t.ShortDesc() }
func (t leetCmd) Exec(ctx context.Context, ev *slack.MessageEvent, info *slack.Info) ([]api.UbotReturn, error) {
	log.Errorf("running leet on: %s", ev.Text)
	ret := make([]api.UbotReturn, 0, 1)
	args, _ := api.GetArgs(ev.Text, t.Name(), info.User.ID)
	leet, err := formatifier.ToLeet(args)
	if err != nil {
		msg := "Unable to leetify"
		ret = append(ret, api.UbotReturn{
			Message:           msg,
			MessageParameters: slack.PostMessageParameters{},
		})
		return ret, nil
	}
	ret = append(ret, api.UbotReturn{
		Message:           leet,
		MessageParameters: slack.PostMessageParameters{},
	})
	return ret, nil
}

type morseCmd string

func (t morseCmd) Name() string      { return string(t) }
func (t morseCmd) Usage() string     { return fmt.Sprintf("Usage: %s <text>", t.Name()) }
func (t morseCmd) ShortDesc() string { return `prints morse code from <text>` }
func (t morseCmd) LongDesc() string  { return t.ShortDesc() }
func (t morseCmd) Exec(ctx context.Context, ev *slack.MessageEvent, info *slack.Info) ([]api.UbotReturn, error) {
	log.Errorf("running morse on: %s", ev.Text)
	ret := make([]api.UbotReturn, 0, 1)
	args, _ := api.GetArgs(ev.Text, t.Name(), info.User.ID)
	morse, err := formatifier.ToMorseCode(args)
	if err != nil {
		msg := "Unable to morse code"
		ret = append(ret, api.UbotReturn{
			Message:           msg,
			MessageParameters: slack.PostMessageParameters{},
		})
		return ret, nil
	}
	ret = append(ret, api.UbotReturn{
		Message:           morse,
		MessageParameters: slack.PostMessageParameters{},
	})
	return ret, nil
}

type leetCmds struct{}

func (t *leetCmds) Init(ctx context.Context) error {
	_, err := api.GetEnv("UBOT_ENABLE_LEET", "")
	if err != nil {
		return err
	}
	log.Info("Loaded leet-pirate plugin")
	return nil
}

func (t *leetCmds) Registry() map[string]api.UbotCommand {
	return map[string]api.UbotCommand{
		"leet":  leetCmd("leet"),
		"morse": morseCmd("morse"),
	}
}

// UbotCommands ...
var UbotCommands leetCmds
