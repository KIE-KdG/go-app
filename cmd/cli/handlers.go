package main

import (
	"bufio"
	"os/exec"
	"strings"
)

type projectForm struct {
	Name string
}

func (app *application) projectPost() {

	form := projectForm{}

	form.Name = "New Project"

	app.projects.Insert(form.Name)

}

func convertModelsToStrings(models []Model) []string {
	var result []string
	for _, model := range models {
		result = append(result, model.Name)
	}
	return result
}

type Model struct {
	Name     string
	ID       string
	Size     string
	Modified string
}

func (app *application) initialllm() cliModel {
	cmd := exec.Command("ollama", "list")
	out, err := cmd.Output()
	if err != nil {
		return cliModel{}
	}


	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var models []Model

	// Skip the header line
	if scanner.Scan() {
		// header: "NAME               ID              SIZE      MODIFIED"
	}

	// Process each subsequent line
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		// Ensure we have at least 7 fields to parse the line correctly
		if len(fields) < 7 {
			continue
		}
		// Combine the size fields (e.g., "2.0" and "GB")
		size := fields[2] + " " + fields[3]
		// Join the remaining fields as the modified timestamp
		modified := strings.Join(fields[4:], " ")

		model := Model{
		Name:     fields[0],
		Size:     size,
		Modified: modified,
		}
		models = append(models, model)
	}

	return cliModel{
		choices:  convertModelsToStrings(models),
		selected: make(map[int]struct{}),
	}
}