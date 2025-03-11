package main

import (
	"kdg/be/lab/internal/validator"
	"net/http"
)

type adminPanelForm struct {
	Chunk               bool   `form:"chunk"`
	ChunkMethod         string `form:"chunkMethod"`
	ChunkCount          string `form:"chunkCount"`
	validator.Validator `form:"-"`
}

func (app *application) adminPanel(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = adminPanelForm{}
	app.render(w, http.StatusOK, "admin.tmpl.html", data)
}

func (app *application) uploadPost(w http.ResponseWriter, r *http.Request) {
	var form adminPanelForm

	if err := app.decodePostForm(r, &form); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	r.ParseMultipartForm(10 << 20)

	if !app.processFormValidation(w, r, form, "admin.tmpl.html") {
		return
	}

	file, handler, err := r.FormFile("myFile")
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	defer file.Close()

	app.infoLog.Printf("Uploaded File: %+v\n", handler.Filename)
	app.infoLog.Printf("File Size: %+v\n", handler.Size)
	app.infoLog.Printf("MIME Header: %+v\n", handler.Header)
	app.infoLog.Printf("Form: %+v\n", form)

	http.Redirect(w, r, "/panel", http.StatusSeeOther)
}