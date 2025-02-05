package main

import (
	"encoding/json"
	"html/template"
	"net/http"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path != "/" {
    app.notFound(w)
    return
  }

  files := []string{
    "ui/html/pages/index.tmpl.html",
  }

  ts, err := template.ParseFiles(files...)
  if err != nil {
    app.serverError(w, err)
    return
  }

  err = ts.ExecuteTemplate(w, "main", nil)
  if err != nil {
    app.serverError(w, err)
  }
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

func (app *application) chatHandler(w http.ResponseWriter, r *http.Request) {
  if r.Method != http.MethodPost {
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.clientError(w, http.StatusNotAcceptable)
		return
	}

	promptResponse, err := app.models.PromptOllama(req.Message)
	if err != nil {
		app.serverError(w, err)
		return
	}

	res := ChatResponse{Response: promptResponse}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}