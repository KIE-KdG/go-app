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
	router.Handler(http.MethodGet, "/api/geojson", protected.ThenFunc(app.geoJsonHandler))
	router.Handler(http.MethodGet, "/ws/chat/:id", chatIDMiddleware(protected.ThenFunc(app.handleConnections)))
	router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(app.userLogoutPost))

	//TODO add roles so that only admins can do following tasks
	router.Handler(http.MethodGet, "/projects", protected.ThenFunc(app.projectsOverview))
	router.Handler(http.MethodPost, "/project/create", protected.ThenFunc(app.projectCreate))
	router.Handler(http.MethodGet, "/project/{id}", protected.ThenFunc(app.projectView))
	router.Handler(http.MethodGet, "/admin/upload", protected.ThenFunc(app.adminUploadForm))

	router.Handler(http.MethodGet, "/panel", protected.ThenFunc(app.adminPanel))
	router.Handler(http.MethodPost, "/project/{id}/sync", protected.ThenFunc(app.syncProjectWithExternal))
	//router.Handler(http.MethodPost, "/api/upload", protected.ThenFunc(app.uploadPost))
	router.Handler(http.MethodGet, "/ws/upload", chatIDMiddleware(protected.ThenFunc(app.handleFileUpload)))
	router.Handler(http.MethodGet, "/ws/process/{id}", chatIDMiddleware(protected.ThenFunc(app.handleDocumentProcessing)))

	standard := alice.New(app.recoverPanic, app.logRequest)

	return standard.Then(router)
}
