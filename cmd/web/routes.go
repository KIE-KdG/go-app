package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {

	router := httprouter.New()

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	fileServer := http.FileServer(http.Dir("./ui/static"))
	router.Handler(http.MethodGet, "/static/*filepath", http.StripPrefix("/static", fileServer))

	dynamic := alice.New(app.sessionManager.LoadAndSave)

  router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/map", dynamic.ThenFunc(app.mapView))
	router.Handler(http.MethodGet, "/socket", dynamic.ThenFunc(app.socketView))
	router.Handler(http.MethodPost, "/api/chat", dynamic.ThenFunc(app.chatHandler))
	router.Handler(http.MethodPost, "/api/geojson", dynamic.ThenFunc(app.geoJsonHandler))

	router.Handler(http.MethodGet, "/ws", dynamic.ThenFunc(app.handleConnections))

	standard := alice.New(app.recoverPanic, app.logRequest)

	return standard.Then(router)
}