package main

import (
	"kdg/be/lab/internal/validator"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

type projectCreateForm struct {
	Name string `form:"name"`
	validator.Validator `form:"-"`
}

func (app *application) projectCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = projectCreateForm{}

	app.render(w, http.StatusOK, "project_create.tmpl.html", data)
}

// projectCreate handles creation of a new project
func (app *application) projectCreatePost(w http.ResponseWriter, r *http.Request) {
	var form projectCreateForm

	// Parse the form
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.errorLog.Print(err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")

	if !form.Valid() {
		app.renderFormErrors(w, r, form, "admin.tmpl.html")
		return
	}

	// Get user ID from session
	userID := app.userIdFromSession(r)
	if userID == uuid.Nil {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	_, err = app.externalAPI.CreateExternalProject(userID.String(), form.Name)
	if err != nil {
		app.errorLog.Printf("Failed to create project in external system: %v", err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Project created successfully")

	http.Redirect(w, r, "/panel", http.StatusSeeOther)
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

	// Get files for this project
	// files, err := app.files.GetByProject(projectID)
	// if err != nil {
	// 	app.serverError(w, err)
	// 	return
	// }

	data := app.newTemplateData(r)
	data.Project = project


	app.render(w, http.StatusOK, "project.tmpl.html", data)
}
