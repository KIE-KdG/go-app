package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"

	"kdg/be/lab/internal/models"
	"kdg/be/lab/internal/validator"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.notFound(w)
		return
	}

	data := app.newTemplateData(r)

	app.render(w, http.StatusOK, "home.tmpl.html", data)
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

type WebSocketRequest struct {
	Message  string `json:"message"`
	DBUsed   bool   `json:"dbUsed"`
	DocsUsed bool   `json:"docsUsed"`
}

type WebSocketResponse struct {
	Prompt string `json:"prompt"`
}

func (app *application) handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.serverError(w, err)
		return
	}
	defer ws.Close()

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			app.errorLog.Println("read error:", err)
			break
		}

		var req WebSocketRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			app.errorLog.Println("unmarshal error:", err)
			continue
		}
		app.infoLog.Printf("Received message: %s, DB: %t, Docs: %t", req.Message, req.DBUsed, req.DocsUsed)

		promptResponse, err := app.chatPort.ForwardMessage(req.Message)
		if err != nil {
			app.serverError(w, err)
			return
		}

		res := WebSocketResponse{Prompt: promptResponse}
		jsonRes, err := json.Marshal(res)
		if err != nil {
			app.serverError(w, err)
			return
		}

		if err := ws.WriteMessage(websocket.TextMessage, jsonRes); err != nil {
			app.errorLog.Println("write error:", err)
			break
		}
	}
}

type userSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, http.StatusOK, "signup.tmpl.html", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "signup.tmpl.html", data)
		return
	}
	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "signup.tmpl.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your signup was succesfull. Please log in.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, http.StatusOK, "login.tmpl.html", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form userLoginForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "login.tmpl.html", data)
		return
	}

	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password incorrect")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "login.tmpl.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	app.sessionManager.Put(r.Context(), "flash", "You have been logger out succesfully")

	http.Redirect(w,r, "/", http.StatusSeeOther)
}

func (app *application) adminPanel(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = adminPanelForm{}
	app.render(w, http.StatusOK, "admin.tmpl.html", data)
}

type adminPanelForm struct {
	Chunk               bool   `form:"chunk"`
	ChunkMethod         string `form:"chunkMethod"`
	ChunkCount          string `form:"chunkCount"`
	validator.Validator `form:"-"`
}

func (app *application) uploadPost(w http.ResponseWriter, r *http.Request) {
	var form adminPanelForm

	if err := app.decodePostForm(r, &form); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	r.ParseMultipartForm(10 << 20)

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "admin.tmpl.html", data)
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
