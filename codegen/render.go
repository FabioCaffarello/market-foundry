package main

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
)

// TemplateData is the data passed to every codegen template.
type TemplateData struct {
	Spec    *FamilySpec
	Derived DerivedFields
}

// RenderArtifact loads the named template and renders it against the spec.
// artifactName must be one of: consumer_spec, pipeline_entry.
func RenderArtifact(spec *FamilySpec, artifactName string, templatesDir string) (string, error) {
	templateFile := artifactName + ".go.tmpl"
	templatePath := templatesDir + "/" + templateFile

	tmplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", templatePath, err)
	}

	tmpl, err := template.New(templateFile).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", templateFile, err)
	}

	data := TemplateData{
		Spec:    spec,
		Derived: spec.Derived(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", templateFile, err)
	}

	return buf.String(), nil
}

// SupportedArtifacts returns the artifact names supported by the current engine.
func SupportedArtifacts() []string {
	return []string{"consumer_spec", "pipeline_entry"}
}
