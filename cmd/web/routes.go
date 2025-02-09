package main

import "net/http"

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static"))
  mux.Handle("/static/", http.StripPrefix("/static", fileServer))

  mux.HandleFunc("/", app.home)
	mux.HandleFunc("/api/chat", app.chatHandler)
	mux.HandleFunc("/api/geojson", app.geojsonHandler)

	return mux
}