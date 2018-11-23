package main

import (
	"context"
	"fmt"

	"github.com/BonnierNews/ubot/pkg/api"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
)

type awCmd string

func (t awCmd) Name() string      { return string(t) }
func (t awCmd) Usage() string     { return fmt.Sprintf("Usage: %s <text>", t.Name()) }
func (t awCmd) ShortDesc() string { return `prints aw suggestions` }
func (t awCmd) LongDesc() string  { return t.ShortDesc() }
func (t awCmd) Exec(ctx context.Context, ev *slack.MessageEvent, info *slack.Info) ([]api.UbotReturn, error) {
	log.Errorf("running AW on: %s", ev.Text)
	ret := make([]api.UbotReturn, 0, 1)
	//args, _ := api.GetArgs(ev.Text, t.Name(), info.User.ID)
	ret = append(ret, api.UbotReturn{
		Message:           "I kväll?",
		MessageParameters: slack.PostMessageParameters{},
	})
	return ret, nil
}

type awCmds struct{}

func (t *awCmds) Init(ctx context.Context) error {
	log.Info("Loaded aw plugin")
	return nil
}

func (t *awCmds) Registry() map[string]api.UbotCommand {
	return map[string]api.UbotCommand{
		"aw": awCmd("aw"),
	}
}

// UbotCommands ...
var UbotCommands awCmds
