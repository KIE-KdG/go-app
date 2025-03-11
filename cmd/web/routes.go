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

	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf)

	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))
	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))

	protected := dynamic.Append(app.requireAuthentication)

  router.Handler(http.MethodGet, "/", protected.ThenFunc(app.home))
	router.Handler(http.MethodPost, "/chat", protected.ThenFunc(app.newChatPost))
	router.Handler(http.MethodGet, "/chat/:id", protected.ThenFunc(app.chat))
	router.Handler(http.MethodGet, "/map", protected.ThenFunc(app.mapView))
	router.Handler(http.MethodPost, "/api/geojson", protected.ThenFunc(app.geoJsonHandler))
	router.Handler(http.MethodGet, "/ws/:id", chatIDMiddleware(protected.ThenFunc(app.handleConnections)))
	router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(app.userLogoutPost))

	//TODO add roles so that only admins can do following tasks
	router.Handler(http.MethodGet, "/panel", protected.ThenFunc(app.adminPanel))
	router.Handler(http.MethodPost, "/api/upload", protected.ThenFunc(app.uploadPost))


	standard := alice.New(app.recoverPanic, app.logRequest)

	return standard.Then(router)
}