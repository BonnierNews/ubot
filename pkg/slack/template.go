package slack

import (
	"fmt"
	"html/template"
)

var (
	FuncMap = template.FuncMap{
		"code": code,
	}
)

func code(v string) string {
	return fmt.Sprintf("`%s`", v)
}
