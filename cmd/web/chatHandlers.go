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

	// Retrieve user's chats
	chats, err := app.chats.RetrieveByUserId(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Retrieve messages for this chat
	messages, err := app.messages.GetByChatID(uuid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Fetch user's projects for the dropdown
	projects, err := app.projects.GetByUserID(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.infoLog.Print(chats)
	app.infoLog.Print(messages)

	data := app.newTemplateData(r)
	data.Chats = chats
	data.Messages = messages
	data.Projects = projects
	data.UserID = userID.String() // Pass user ID to template for JavaScript

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

	// Get the dummy GeoJSON data
	app.infoLog.Printf("Retrieving GeoJSON data...")
	geoJsonMap, err := app.geoData.Dummy()
	if err != nil {
		app.errorLog.Printf("Error loading GeoJSON: %v", err)
		app.serverError(w, fmt.Errorf("error loading GeoJSON data: %w", err))
		return
	}

	// Debug the map content
	app.infoLog.Printf("GeoJSON data type: %T", geoJsonMap)
	if geoJsonMap == nil {
		app.errorLog.Printf("GeoJSON data is nil")
		// Provide a simple valid GeoJSON as fallback
		geoJsonMap = map[string]interface{}{
			"type": "FeatureCollection",
			"features": []interface{}{},
		}
	}

	// Use our debug function to convert and validate the GeoJSON
	geoJsonString, err := app.debugGeoJSON(geoJsonMap)
	if err != nil {
		app.errorLog.Printf("Error processing GeoJSON: %v", err)
		app.serverError(w, fmt.Errorf("error processing GeoJSON: %w", err))
		return
	}

	// Get user ID for retrieving chats if authenticated
	userID := app.userIdFromSession(r)
	
	// Get chat history for the authenticated user
	chats, err := app.chats.RetrieveByUserId(userID)
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.GeoData = geoJsonString  // Pass as string instead of map
	data.Chats = chats

	app.infoLog.Printf("Rendering map template with GeoJSON data")
	app.render(w, http.StatusOK, "map.tmpl.html", data)
}

func (app *application) geoJsonHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}

	// Get the dummy GeoJSON data
	geoJsonMap, err := app.geoData.Dummy()
	if err != nil {
		app.serverError(w, fmt.Errorf("error loading GeoJSON data: %w", err))
		return
	}

	app.writeJSON(w, http.StatusOK, geoJsonMap)
}