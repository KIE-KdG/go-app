package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justinas/nosurf"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace)

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) newTemplateData(r *http.Request) *templateData {
	return &templateData{
		CurrentYear:     time.Now().Year(),
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
		IsAuthenticated: app.isAuthenticated(r),
		CSRFToken:       nosurf.Token(r),
	}
}

func (app *application) render(w http.ResponseWriter, status int, page string, data *templateData) {
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, err)
		return
	}

	buf := new(bytes.Buffer)

	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(status)

	buf.WriteTo(w)
}

func (app *application) decodePostForm(r *http.Request, dst any) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}

		return err
	}

	return nil
}

func (app *application) isAuthenticated(r *http.Request) bool {
	return app.sessionManager.Exists(r.Context(), "authenticatedUserID")
}

func (app *application) userIdFromSession(r *http.Request) uuid.UUID {
	uuidStr := app.sessionManager.GetString(r.Context(), "authenticatedUserID")
	id, err := uuid.Parse(uuidStr)
	if err != nil {
			return uuid.Nil
	}
	return id
}
// writeJSON encodes an interface to a JSON response
func (app *application) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	// Set the Content-Type header
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	// Encode the data to JSON
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		app.serverError(w, err)
	}
}

// readJSON decodes a JSON request body into a destination struct
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	// Limit the request body size
	r.Body = http.MaxBytesReader(w, r.Body, 1048576) // 1MB limit

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(dst)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return err
	}

	return nil
}

// debugGeoJSON ensures we're generating valid JSON for the template
func (app *application) debugGeoJSON(geoJsonMap map[string]interface{}) (string, error) {
	// Marshal the map to pretty JSON 
	jsonBytes, err := json.MarshalIndent(geoJsonMap, "", "  ")
	if err != nil {
		return "", err
	}

	// Convert to string
	jsonStr := string(jsonBytes)

	// Log a sample of the JSON for debugging
	if len(jsonStr) > 100 {
		app.infoLog.Printf("JSON sample (first 100 chars): %s...", jsonStr[:100])
	} else {
		app.infoLog.Printf("JSON sample: %s", jsonStr)
	}

	// Validate the JSON by attempting to parse it back
	var validate interface{}
	if err := json.Unmarshal(jsonBytes, &validate); err != nil {
		app.errorLog.Printf("Generated invalid JSON: %v", err)
		return "", fmt.Errorf("generated invalid JSON: %w", err)
	}

	return jsonStr, nil
}

// parseUUID parses a UUID string and handles errors
func (app *application) parseUUID(w http.ResponseWriter, idStr string) (uuid.UUID, bool) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		app.notFound(w)
		return uuid.UUID{}, false
	}
	return id, true
}

// renderFormErrors renders a template with form validation errors
func (app *application) renderFormErrors(w http.ResponseWriter, r *http.Request, form interface{}, templateName string) {
	data := app.newTemplateData(r)
	data.Form = form
	app.render(w, http.StatusUnprocessableEntity, templateName, data)
}

// processFormValidation handles common form validation pattern
func (app *application) processFormValidation(w http.ResponseWriter, r *http.Request, form interface{}, templateName string) bool {
	if validatable, ok := form.(interface{ Valid() bool }); ok {
		if !validatable.Valid() {
			app.renderFormErrors(w, r, form, templateName)
			return false
		}
	}
	return true
}

// setFlashMessage sets a flash message in the session and redirects
func (app *application) setFlashAndRedirect(w http.ResponseWriter, r *http.Request, flashMessage, redirectURL string, status int) {
	app.sessionManager.Put(r.Context(), "flash", flashMessage)
	http.Redirect(w, r, redirectURL, status)
}
