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

	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))
	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))

	protected := dynamic.Append(app.requireAuthentication)

  router.Handler(http.MethodGet, "/", protected.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/map", protected.ThenFunc(app.mapView))
	router.Handler(http.MethodGet, "/socket", protected.ThenFunc(app.socketView))
	router.Handler(http.MethodPost, "/api/chat", protected.ThenFunc(app.chatHandler))
	router.Handler(http.MethodPost, "/api/geojson", protected.ThenFunc(app.geoJsonHandler))

	router.Handler(http.MethodGet, "/ws", protected.ThenFunc(app.handleConnections))

	standard := alice.New(app.recoverPanic, app.logRequest)

	return standard.Then(router)
}