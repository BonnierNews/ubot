package api

import (
	"context"

	"github.com/nlopes/slack"
)

// Symbolname for exported symbols
const (
	SymbolName = "UbotCommands"
)

// UbotReturn a slice of Slack messages
type UbotReturn struct {
	Message           string
	MessageParameters slack.PostMessageParameters
}

// UbotModule a plugin that can be initialized
type UbotModule interface {
	Init(context.Context) error
}

// UbotCommand represents a bot command, all ubot commands must implement this
type UbotCommand interface {
	Name() string
	Usage() string
	ShortDesc() string
	LongDesc() string
	Exec(context.Context, *slack.MessageEvent, *slack.Info) ([]UbotReturn, error)
}

// UbotCommands a plugin that contains one or more command
type UbotCommands interface {
	UbotModule
	Registry() map[string]UbotCommand
}
