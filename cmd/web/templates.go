package main

import (
	"fmt"
	"html/template"
	"kdg/be/lab/internal/models"
	"path/filepath"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type templateData struct {
	Completion      string
	CurrentYear     int
	Chats           []*models.Chat
	Messages        []*models.Message
	GeoData         string
	Form            any
	Flash           string
	IsAuthenticated bool
	CSRFToken       string
	Localizer       *i18n.Localizer
	Projects        []*models.Project
	Project         *models.Project
	ProjectDatabase *models.ProjectDatabase
	Files           []*models.File
	HasDocuments    bool
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format("02 Jan 2006 at 15:04")
}

var functions = template.FuncMap{
	"humanDate": humanDate,
	"formatFileSize": formatFileSize,
	"roleBadgeClass": roleBadgeClass,
	"statusBadgeClass": statusBadgeClass,
}

// Role badge helper function
func roleBadgeClass(role string) string {
	switch role {
	case "CONTENT":
			return "badge badge-primary"
	case "METADATA":
			return "badge badge-secondary"
	case "SCHEMA":
			return "badge badge-accent"
	default:
			return "badge badge-ghost"
	}
}

// Status badge helper function
func statusBadgeClass(status string) string {
	switch status {
	case "processed":
			return "badge badge-success"
	case "uploaded":
			return "badge badge-info"
	case "error":
			return "badge badge-error"
	default:
			return "badge badge-warning"
	}
}

// Add this function
func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}

// List of templates that should use the auth_base.tmpl.html instead of base.tmpl.html
var authTemplates = map[string]bool{
	"login.tmpl.html":  true,
	"signup.tmpl.html": true,
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := filepath.Glob("./ui/html/pages/*.tmpl.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		
		// Determine which base template to use
		var ts *template.Template
		
		if authTemplates[name] {
			// For authentication pages, use auth_base.tmpl.html
			ts, err = template.New(name).Funcs(functions).ParseFiles("./ui/html/auth_base.tmpl.html")
		} else {
			// For regular pages, use the standard base.tmpl.html
			ts, err = template.New(name).Funcs(functions).ParseFiles("./ui/html/base.tmpl.html")
		}
		
		if err != nil {
			return nil, err
		}

		// Parse partials (shared by both base templates)
		ts, err = ts.ParseGlob("./ui/html/partials/*.tmpl.html")
		if err != nil {
			return nil, err
		}

		// Parse the page template
		ts, err = ts.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}
	return cache, nil
}