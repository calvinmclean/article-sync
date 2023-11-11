package main

import (
	"fmt"
	"html/template"
	"io"
	"os"
)

const (
	commentTemplate = `## Article Sync Summary

After merge, {{ len .NewArticles }} new article will be created and {{ len .UpdatedArticles }} existing article will be updated.

{{- if gt (len .NewArticles) 0 }}

### New Articles
{{- range .NewArticles }}
- {{ .Title }}
{{- end }}
{{- end }}
{{- if gt (len .UpdatedArticles) 0 }}

### Updated Articles
{{- range .UpdatedArticles }}
- [{{ .Title }}]({{ .URL }})
{{- end }}
{{- end }}`

	commitTemplate = `completed sync: {{ len .NewArticles }} new, {{ len .UpdatedArticles }} updated
{{ if or (gt (len .NewArticles) 0) (gt (len .UpdatedArticles) 0) }}{{ end }}
{{- range .NewArticles }}
- new: {{ .Title }} ({{ .URL }})
{{- end }}
{{- range .UpdatedArticles }}
- updated: {{ .Title }} ({{ .URL }})
{{- end }}`
)

func renderTemplateToFile(path, tmpl string, data commentData) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	err = renderTemplate(tmpl, data, file)
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

func renderTemplate(tmplString string, data commentData, destination io.Writer) error {
	tmpl := template.Must(template.New("tmpl").Parse(tmplString))

	err := tmpl.Execute(destination, data)
	if err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	return nil
}
