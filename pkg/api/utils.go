package api

import (
	"fmt"
	"strings"
)

// GetArgs returns arguments
func GetArgs(line string, cmd string, id string) (string, error) {
	prefix := fmt.Sprintf("<@%s> ", id)
	args := strings.TrimPrefix(line, prefix)
	args = strings.TrimPrefix(args, cmd)
	args = strings.TrimSpace(args)
	return args, nil
}
