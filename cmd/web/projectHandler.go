package main

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"kdg/be/lab/internal/models"
	"kdg/be/lab/internal/validator"
	"net/http"
)

type projectCreateForm struct {
	Name                string `form:"name"`
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

	_, err = app.externalAPI.CreateExternalProject(userID, form.Name)
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
	files, err := app.files.GetByProject(projectID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Get database for this project
	// We'll handle the case where no database exists
	var projectDatabase *models.ProjectDatabase
	var projectSchemas []string

	projectDatabase, err = app.projectDatabase.GetByProjectID(projectID)
	if err != nil {
		if !errors.Is(err, models.ErrNoRecord) {
			// Only return a server error if it's not a "no record" error
			app.serverError(w, err)
			return
		}
		// If there's no database record, projectDatabase will be nil
	}

	// Only try to fetch schemas if we have a database
	if projectDatabase != nil && projectDatabase.ID != uuid.Nil {
		schemasPtr, err := app.externalAPI.GetDatabaseSchemas(projectDatabase.ID)
		if err != nil {
			app.errorLog.Printf("Schema get error: %v", err)
			app.errorLog.Printf("Continuing with empty schema list")
			// Continue with empty schemas rather than failing the whole page
		} else if schemasPtr != nil {
			projectSchemas = *schemasPtr // Dereference only if not nil
		}
	}

	data := app.newTemplateData(r)
	data.Project = project
	data.Files = files
	data.ProjectDatabase = projectDatabase
	data.ProjectSchemas = projectSchemas // Assign slice directly (not dereferenced pointer)
	data.HasDocuments = len(files) > 0   // Flag to indicate if documents exist
	data.Form = projectForms{}

	app.render(w, http.StatusOK, "project.tmpl.html", data)
}

type projectForms struct {
	DatabaseForm databaseSetupForm
	SchemaForm   schemaSetupForm
}

// Updated database setup form with individual connection parameters
type databaseSetupForm struct {
	ProjectID           string `form:"project_id"`
	DbType              string `form:"dbtype"`
	Server              string `form:"server"`
	Port                string `form:"port"`
	Database            string `form:"database"`
	Username            string `form:"username"`
	Password            string `form:"password"`
	TrustServerCert     bool   `form:"trust_server_cert"`
	validator.Validator `form:"-"`
}

type schemaSetupForm struct {
	Name                []string `form:"name"`
	validator.Validator `form:"-"`
}

// GenerateConnectionString creates the appropriate connection string based on database type
func (form *databaseSetupForm) GenerateConnectionString() (string, error) {
	switch form.DbType {
	case "sqlserver":
		trustServerCert := "no"
		if form.TrustServerCert {
			trustServerCert = "yes"
		}
		return fmt.Sprintf("DRIVER={ODBC Driver 17 for SQL Server};SERVER=%s,%s;DATABASE=%s;UID=%s;PWD=%s;TrustServerCertificate=%s",
			form.Server, form.Port, form.Database, form.Username, form.Password, trustServerCert), nil

	case "mysql", "mariadb":
		return fmt.Sprintf("server=%s;port=%s;database=%s;user=%s;password=%s",
			form.Server, form.Port, form.Database, form.Username, form.Password), nil

	case "postgres":
		sslMode := "disable"
		if !form.TrustServerCert {
			sslMode = "require"
		}
		return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
			form.Server, form.Port, form.Database, form.Username, form.Password, sslMode), nil

	default:
		return "", fmt.Errorf("unsupported database type: %s", form.DbType)
	}
}

// Handler for database connection setup
func (app *application) projectDatabaseSetupPost(w http.ResponseWriter, r *http.Request) {
	// Parse the form
	var parent projectForms

	// extract child from parent
	form := parent.DatabaseForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.errorLog.Printf("Form decode error: %v", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Validate form fields
	form.CheckField(validator.NotBlank(form.ProjectID), "project_id", "Project ID is required")
	form.CheckField(validator.NotBlank(form.DbType), "dbtype", "Database type is required")
	form.CheckField(validator.NotBlank(form.Server), "server", "Server host is required")
	form.CheckField(validator.NotBlank(form.Port), "port", "Port is required")
	form.CheckField(validator.NotBlank(form.Database), "database", "Database name is required")
	form.CheckField(validator.NotBlank(form.Username), "username", "Username is required")
	form.CheckField(validator.NotBlank(form.Password), "password", "Password is required")

	// Parse project ID
	projectID, err := uuid.Parse(form.ProjectID)
	if err != nil {
		app.errorLog.Printf("Invalid project ID: %v", err)
		form.AddNonFieldError("Invalid project ID")
		app.renderFormErrors(w, r, form, "project.tmpl.html")
		return
	}

	// If there are validation errors, re-render the form
	if !form.Valid() {
		// Get the project and files for the template
		project, pErr := app.projects.Get(projectID)
		if pErr != nil {
			app.errorLog.Printf("Project not found: %v", pErr)
			app.notFound(w)
			return
		}

		files, fErr := app.files.GetByProject(projectID)
		if fErr != nil {
			app.errorLog.Printf("Error fetching files: %v", fErr)
			app.serverError(w, fErr)
			return
		}

		// Prepare template data
		data := app.newTemplateData(r)
		data.Project = project
		data.Files = files
		data.HasDocuments = len(files) > 0
		data.Form = form // Include the form with errors

		// Render the template
		app.render(w, http.StatusUnprocessableEntity, "project.tmpl.html", data)
		return
	}

	// Generate connection string on the backend
	connectionString, err := form.GenerateConnectionString()
	if err != nil {
		app.errorLog.Printf("Failed to generate connection string: %v", err)
		form.AddNonFieldError(fmt.Sprintf("Database configuration error: %v", err))
		app.renderFormErrors(w, r, form, "project.tmpl.html")
		return
	}

	// Save the database configuration
	// All validation passed, create database connection
	_, err = app.externalAPI.CreateProjectDatabase(projectID, connectionString, form.DbType)
	if err != nil {
		app.errorLog.Printf("Database connection creation error: %v", err)
		app.serverError(w, err)
		return
	}

	// Set success flash message
	app.sessionManager.Put(r.Context(), "flash", "Database connection successfully created")

	// Redirect back to the project page
	http.Redirect(w, r, fmt.Sprintf("/project/view/%s", projectID), http.StatusSeeOther)
}

type schemaCreate struct {
	DbID                string   `form:"db_id"`
	ProjectID           string   `form:"project_id"`
	SchemaName          []string `form:"schemaName"`
	validator.Validator `form:"-"`
}

func (app *application) databaseSchemaPost(w http.ResponseWriter, r *http.Request) {
	var schemaForm schemaCreate
	err := app.decodePostForm(r, &schemaForm)
	if err != nil {
		app.errorLog.Printf("Form decode error: %v", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	schemaForm.CheckField(len(schemaForm.SchemaName) > 0, "schemaName", "Please select at least one schema")

	projectID, err := uuid.Parse(schemaForm.ProjectID)
	if err != nil {
		app.errorLog.Printf("Invalid project ID: %v", err)
		schemaForm.AddNonFieldError("Invalid project ID")
		app.renderSchemaFormWithErrors(w, r, projectID, schemaForm)
		return
	}

	// Parse database ID
	dbID, err := uuid.Parse(schemaForm.DbID)
	if err != nil {
		app.errorLog.Printf("Invalid database ID: %v", err)
		schemaForm.AddNonFieldError("Invalid database ID")
		app.renderSchemaFormWithErrors(w, r, projectID, schemaForm)
		return
	}

	if !schemaForm.Valid() {
		app.renderSchemaFormWithErrors(w, r, projectID, schemaForm)
		return
	}

	// Now call the API with both the database ID and schema name
	_, err = app.externalAPI.CreateDatabaseSchema(dbID, schemaForm.SchemaName)
	if err != nil {
		app.errorLog.Printf("Database schema creation error: %v", err)
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Schema successfully created")
	http.Redirect(w, r, fmt.Sprintf("/project/view/%s", projectID), http.StatusSeeOther)
}

// Helper function to avoid code duplication when rendering the schema form with errors
func (app *application) renderSchemaFormWithErrors(w http.ResponseWriter, r *http.Request, projectID uuid.UUID, schemaForm schemaCreate) {
	// Get the project and files for the template
	project, pErr := app.projects.Get(projectID)
	if pErr != nil {
		app.errorLog.Printf("Project not found: %v", pErr)
		app.notFound(w)
		return
	}

	files, fErr := app.files.GetByProject(projectID)
	if fErr != nil {
		app.errorLog.Printf("Error fetching files: %v", fErr)
		app.serverError(w, fErr)
		return
	}

	// Get project database
	projectDatabase, dbErr := app.projectDatabase.GetByProjectID(projectID)
	if dbErr != nil && !errors.Is(dbErr, models.ErrNoRecord) {
		app.errorLog.Printf("Error fetching project database: %v", dbErr)
		app.serverError(w, dbErr)
		return
	}

	// Create the correct form structure that matches template expectations
	formData := projectForms{
		SchemaForm: schemaSetupForm{
			Name: schemaForm.SchemaName,
		},
	}

	// Copy validation errors
	formData.SchemaForm.FieldErrors = schemaForm.FieldErrors
	formData.SchemaForm.NonFieldErrors = schemaForm.NonFieldErrors

	// Prepare template data
	data := app.newTemplateData(r)
	data.Project = project
	data.Files = files
	data.ProjectDatabase = projectDatabase
	data.HasDocuments = len(files) > 0
	data.Form = formData

	// Render the template
	app.render(w, http.StatusUnprocessableEntity, "project.tmpl.html", data)
}
