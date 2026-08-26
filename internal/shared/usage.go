package shared

import (
	"bytes"
	"strings"
	"text/template"
)

// UsageData contains the fields rendered by TaskUsage.
type UsageData struct {
	Synopsis string
	AppName  string
	Command  string
	Body     string
}

const usageTemplate = `{{.Synopsis}}

USAGE:

  {{.AppName}} {{.Command}} [FLAGS]
{{if .Body}}
{{.Body}}{{end}}
`

// TaskUsage renders a consistent usage block for a CLI task.
func TaskUsage(data UsageData) string {
	tmpl := template.Must(template.New("usage").Parse(usageTemplate))
	data.Body = strings.Trim(data.Body, "\n")
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return data.Synopsis
	}
	return out.String()
}
