package main

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	app.newChatPost(w, r)
}

func (app *application) chat(w http.ResponseWriter, r *http.Request) {
	userID := app.userIdFromSession(r)
	params := httprouter.ParamsFromContext(r.Context())

	idStr := params.ByName("id")

	uuid, ok := app.parseUUID(w, idStr)
	if !ok {
		return
	}

	chats, err := app.chats.RetrieveByUserId(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	messages, err := app.messages.GetByChatID(uuid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.infoLog.Print(chats)
	app.infoLog.Print(messages)

	data := app.newTemplateData(r)
	data.Chats = chats
	data.Messages = messages

	app.render(w, http.StatusOK, "home.tmpl.html", data)
}

func (app *application) newChatPost(w http.ResponseWriter, r *http.Request) {
	userID := app.userIdFromSession(r)

	chatID, err := app.chats.Insert(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/chat/%s", chatID), http.StatusSeeOther)
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

	app.writeJSON(w, http.StatusOK, geoData)
}