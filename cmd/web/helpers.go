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

func (app *application) userIdFromSession(r *http.Request) int {
	return app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
}

// New helper functions

// writeJSON writes a JSON response with the specified status code and data
func (app *application) writeJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Add any custom headers
	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(jsonData)

	return nil
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

// sendWSJSON marshals and sends data as a WebSocket TextMessage
func sendWSJSON(ws *websocket.Conn, data interface{}) error {
	jsonRes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ws.WriteMessage(websocket.TextMessage, jsonRes)
}