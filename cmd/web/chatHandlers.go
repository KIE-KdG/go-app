package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	app.newChatPost(w, r)
}

func (app *application) chat(w http.ResponseWriter, r *http.Request) {
	userID := app.userIdFromSession(r)
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	chats, err := app.chats.RetrieveByUserId(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	messages, err := app.messages.GetByChatID(id)
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.Chats = chats
	data.Messages = messages

	app.render(w, http.StatusOK, "home.tmpl.html", data)
}

func (app *application) newChatPost(w http.ResponseWriter, r *http.Request) {
	userID := app.userIdFromSession(r)

	id, err := app.chats.Insert(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/chat/%d", id), http.StatusSeeOther)
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

	jsonData, err := json.Marshal(req)
	if err != nil {
		app.serverError(w, err)
		return
	}

	promptResponse, err := app.chatPort.ForwardMessage(string(jsonData))
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
