package main

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

// projectsOverview shows all projects for the current user
func (app *application) projectsOverview(w http.ResponseWriter, r *http.Request) {
	// Get user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	
	// Get all projects for this user
	projects, err := app.projects.GetByUserID(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}
	
	data := app.newTemplateData(r)
	data.Projects = projects
	
	app.render(w, http.StatusOK, "projects.tmpl.html", data)
}

// projectCreate handles creation of a new project
func (app *application) projectCreate(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			app.clientError(w, http.StatusMethodNotAllowed)
			return
	}
	
	// Parse the form
	err := r.ParseForm()
	if err != nil {
			app.clientError(w, http.StatusBadRequest)
			return
	}
	
	// Get the form values
	name := r.PostForm.Get("name")
	
	// Basic validation
	if name == "" {
			app.clientError(w, http.StatusBadRequest)
			return
	}
	
	// Get user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
			app.clientError(w, http.StatusUnauthorized)
			return
	}
	
	_, err = app.externalAPI.CreateExternalProject(userID.String(), name)
	if err != nil {
			app.errorLog.Printf("Failed to create project in external system: %v", err)
			return
	}

	app.sessionManager.Put(r.Context(), "flash", "Project created successfully")

	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

// projectView shows details for a specific project
func (app *application) projectView(w http.ResponseWriter, r *http.Request) {
	// Get project ID from URL
	params := httprouter.ParamsFromContext(r.Context())
	projectIDStr := params.ByName("id")
	
	// Parse the UUID
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		app.notFound(w)
		return
	}
	
	// Get the project
	project, err := app.projects.Get(projectID)
	if err != nil {
		app.notFound(w)
		return
	}
	
	// Get the user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	
	// Check if user owns this project
	if project.UserID != userID {
		app.clientError(w, http.StatusForbidden)
		return
	}
	
	// Get files for this project
	files, err := app.files.GetByProject(projectID)
	if err != nil {
		app.serverError(w, err)
		return
	}
	
	data := app.newTemplateData(r)
	data.Project = project
	data.Files = files
	
	app.render(w, http.StatusOK, "project.tmpl.html", data)
}

// adminUploadForm shows the upload form with project selection
func (app *application) adminUploadForm(w http.ResponseWriter, r *http.Request) {
	// Get user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}
	
	// Get all projects for this user
	projects, err := app.projects.GetByUserID(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}
	
	// Check if a project was specified in the URL
	projectIDStr := r.URL.Query().Get("project")
	if projectIDStr != "" {
		projectID, err := uuid.Parse(projectIDStr)
		if err == nil {
			// Verify this project belongs to the user
			project, err := app.projects.Get(projectID)
			if err == nil && project.UserID == userID {
				data := app.newTemplateData(r)
				data.Projects = projects
				data.SelectedProject = project
				
				app.render(w, http.StatusOK, "admin.tmpl.html", data)
				return
			}
		}
	}
	
	// If no valid project was specified, just show the form with all projects
	data := app.newTemplateData(r)
	data.Projects = projects
	
	app.render(w, http.StatusOK, "admin.tmpl.html", data)
}

func (app *application) syncProjectWithExternal(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from the URL
	params := httprouter.ParamsFromContext(r.Context())
	projectIDStr := params.ByName("id")
	
	// Parse the UUID
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
			app.notFound(w)
			return
	}
	
	// Get the project
	project, err := app.projects.Get(projectID)
	if err != nil {
			app.notFound(w)
			return
	}
	
	// Verify the user owns this project
	userID := app.userIdFromSession(r)
	if project.UserID != userID {
			app.clientError(w, http.StatusForbidden)
			return
	}
	
	// Check if already synced
	if project.ExternalID != "" {
			app.sessionManager.Put(r.Context(), "flash", "Project is already synchronized")
			http.Redirect(w, r, fmt.Sprintf("/project/%s", projectID), http.StatusSeeOther)
			return
	}
	
	// Attempt to create in external system
	externalProjectResp, err := app.externalAPI.CreateExternalProject(userID.String(), project.Name)
	if err != nil {
			app.errorLog.Printf("Failed to sync project with external system: %v", err)
			app.sessionManager.Put(r.Context(), "flash", "Failed to sync project with external system. Please try again later.")
			http.Redirect(w, r, fmt.Sprintf("/project/%s", projectID), http.StatusSeeOther)
			return
	}
	
	// Update our local project with the external ID
	err = app.projects.UpdateExternalID(projectID, externalProjectResp.ProjectID)
	if err != nil {
			app.errorLog.Printf("Failed to update project with external ID: %v", err)
			app.sessionManager.Put(r.Context(), "flash", "Project was synced but failed to update local record.")
			http.Redirect(w, r, fmt.Sprintf("/project/%s", projectID), http.StatusSeeOther)
			return
	}
	
	app.sessionManager.Put(r.Context(), "flash", "Project successfully synchronized with external system")
	http.Redirect(w, r, fmt.Sprintf("/project/%s", projectID), http.StatusSeeOther)
}