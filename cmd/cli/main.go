package main

import (
	"flag"
	"log"

	"kdg/be/lab/internal/db"
	"kdg/be/lab/internal/model"
	"kdg/be/lab/internal/models"

	tea "github.com/charmbracelet/bubbletea"
)

type application struct {
	errorLog *log.Logger
	infoLog  *log.Logger
	db       *db.DB
	models   *model.Models
	chatPort *model.ChatPort
	geoData  *models.GeoData
	users    *models.UserModel
	projects *models.ProjectModel
}

type cliModel struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func (m cliModel) Init() tea.Cmd {
	return nil
}

func (m cliModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}
	return m, nil
}

func (m cliModel) View() string {
	s := "Select a model to use:\n\n"

	if len(m.choices) == 0 {
		s += "No models found.\n"
		return s
	}

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		s += cursor + " [" + checked + "] " + choice + "\n"
	}
	s += "\nPress q to quit.\n"
	return s
}

func main() {
	dsn := flag.String("dsn", "data/sqlite_lab.db", "sqlite data source name")
	
	db, err := db.OpenDB(*dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := application{
		projects: &models.ProjectModel{DB: db},
	}


	p := tea.NewProgram(app.initialllm())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

}
