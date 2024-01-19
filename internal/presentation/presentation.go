package presentation

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/CameronXie/go-opa-reviewer/internal/review"
)

type markdownData struct {
	Reviews []string
	Errors  []string
}

func Markdown(results []review.Result) string {
	if len(results) == 0 {
		return "no review results available"
	}

	reviews := make([]string, 0)
	errors := make([]string, 0)

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, markdownListRow(result.File, result.Error.Error()))
			continue
		}

		reviews = append(reviews, markdownListRow(result.File, string(result.Output)))
	}

	outputTmpl := `{{if .Reviews -}}
Reviews:
{{range .Reviews}}{{.}}
{{end}}{{end}}{{if .Errors}}
Errors:
{{range .Errors}}{{.}}
{{end}}{{end}}`

	tmpl := template.Must(template.New("outputTmpl").Parse(outputTmpl))

	var output bytes.Buffer
	_ = tmpl.Execute(&output, markdownData{Reviews: reviews, Errors: errors})

	return output.String()
}

func markdownListRow(file, comment string) string {
	return fmt.Sprintf("* %s: %s", file, comment)
}
