package api

import (
	"fmt"
	"os"
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

// GetEnv returns a environment variable
func GetEnv(key, fallback string) (string, error) {
	if value, ok := os.LookupEnv(key); ok {
		return value, nil
	} else {
		if fallback == "" {
			return "", fmt.Errorf("To use this plugin, set the %s environment variable", key)
		}
	}
	return fallback, nil
}
