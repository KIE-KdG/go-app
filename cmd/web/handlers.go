package main

import (
	"net/http"
	"html/template"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path != "/" {
    app.notFound(w)
    return
  }

  files := []string{
    "ui/html/pages/index.html",
  }

  ts, err := template.ParseFiles(files...)
  if err != nil {
    app.serverError(w, err)
    return
  }

  err = ts.Execute(w, nil)
  if err != nil {
    app.serverError(w, err)
    return
  }
}