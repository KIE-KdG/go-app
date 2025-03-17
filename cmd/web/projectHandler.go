package main

import (
	"errors"
	"fmt"
	"kdg/be/lab/internal/models"
	"kdg/be/lab/internal/validator"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

type projectCreateForm struct {
	Name                string `form:"name"`
	validator.Validator `form:"-"`
}

// RegisteredSchema represents a schema that has been saved in metadata
type RegisteredSchema struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// projectView shows details for a specific project with metadata schemas
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
	var schemaList []string
	var registeredSchemas []RegisteredSchema

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
		// Get all available schemas from the database
		schemasPtr, err := app.externalAPI.GetDatabaseSchemas(projectDatabase.ID)
		if err != nil {
			app.errorLog.Printf("Schema get error: %v", err)
			app.errorLog.Printf("Continuing with empty schema list")
			// Continue with empty schemas rather than failing the whole page
		} else if schemasPtr != nil {
			schemaList = *schemasPtr // Dereference only if not nil
		}

		// Get registered schemas (those with IDs in our metadata)
		dbSchemas, err := app.schemas.ListSchemasByDatabaseID(projectDatabase.ID)
		if err != nil {
			app.errorLog.Printf("Error fetching registered schemas: %v", err)
		} else {
			// Convert to the format needed by the template
			for _, schema := range dbSchemas {
				registeredSchemas = append(registeredSchemas, RegisteredSchema{
					ID:   schema.ID,
					Name: schema.Name,
				})
			}
			app.infoLog.Printf("Found %d registered schemas with IDs", len(registeredSchemas))
		}
	}

	data := app.newTemplateData(r)
	data.Project = project
	data.Files = files
	data.ProjectDatabase = projectDatabase
	data.SchemaList = schemaList
	data.RegisteredSchemas = registeredSchemas
	data.HasDocuments = len(files) > 0
	data.Form = projectForms{} // Initialize empty form

	app.render(w, http.StatusOK, "project.tmpl.html", data)
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
	var dbForm databaseSetupForm

	err := app.decodePostForm(r, &dbForm)
	if err != nil {
		app.errorLog.Printf("Form decode error: %v", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Validate form fields
	dbForm.CheckField(validator.NotBlank(dbForm.ProjectID), "project_id", "Project ID is required")
	dbForm.CheckField(validator.NotBlank(dbForm.DbType), "dbtype", "Database type is required")
	dbForm.CheckField(validator.NotBlank(dbForm.Server), "server", "Server host is required")
	dbForm.CheckField(validator.NotBlank(dbForm.Port), "port", "Port is required")
	dbForm.CheckField(validator.NotBlank(dbForm.Database), "database", "Database name is required")
	dbForm.CheckField(validator.NotBlank(dbForm.Username), "username", "Username is required")
	dbForm.CheckField(validator.NotBlank(dbForm.Password), "password", "Password is required")

	// Parse project ID
	projectID, err := uuid.Parse(dbForm.ProjectID)
	if err != nil {
		app.errorLog.Printf("Invalid project ID: %v", err)
		dbForm.AddNonFieldError("Invalid project ID")
		
		// Create proper form with nested structure
		formData := projectForms{
			DatabaseForm: dbForm,
		}
		
		app.renderFormWithErrors(w, r, projectID, formData)
		return
	}

	// If there are validation errors, re-render the form
	if !dbForm.Valid() {
		// Create proper form with nested structure for the template
		formData := projectForms{
			DatabaseForm: dbForm,
		}
		
		app.renderFormWithErrors(w, r, projectID, formData)
		return
	}

	// Generate connection string on the backend
	connectionString, err := dbForm.GenerateConnectionString()
	if err != nil {
		app.errorLog.Printf("Failed to generate connection string: %v", err)
		dbForm.AddNonFieldError(fmt.Sprintf("Database configuration error: %v", err))
		
		// Create proper form with nested structure
		formData := projectForms{
			DatabaseForm: dbForm,
		}
		
		app.renderFormWithErrors(w, r, projectID, formData)
		return
	}

	// Save the database configuration
	// All validation passed, create database connection
	_, err = app.externalAPI.CreateProjectDatabase(projectID, connectionString, dbForm.DbType)
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

// Common function to render form errors
func (app *application) renderFormWithErrors(w http.ResponseWriter, r *http.Request, projectID uuid.UUID, formData projectForms) {
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

	// Prepare template data
	data := app.newTemplateData(r)
	data.Project = project
	data.Files = files
	data.ProjectDatabase = projectDatabase
	data.HasDocuments = len(files) > 0
	data.Form = formData  // Pass the properly structured form data

	// Render the template
	app.render(w, http.StatusUnprocessableEntity, "project.tmpl.html", data)
}