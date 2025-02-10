package main

import (
	"encoding/json"
	"net/http"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFound(w)
		return
	}

	data := app.newTemplateData(r)

	app.render(w, http.StatusOK, "home.tmpl.html", data)
}

func (app *application) mapView(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/map" {
		app.notFound(w)
		return
	}

	json, err := app.geoData.Dummy()
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.GeoData = json

	app.render(w, http.StatusOK, "map.tmpl.html", data)
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

func (app *application) geoJsonHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}

	geoData, err := app.geoData.Dummy()
	if err != nil {
		app.notFound(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(geoData); err != nil {
		app.serverError(w, err)
		return
	}
}