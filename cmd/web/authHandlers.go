package main

import (
	"errors"
	"kdg/be/lab/internal/models"
	"kdg/be/lab/internal/validator"
	"net/http"
)

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
		app.renderFormErrors(w, r, form, "signup.tmpl.html")
		return
	}
	
	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")
			app.renderFormErrors(w, r, form, "signup.tmpl.html")
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.setFlashAndRedirect(w, r, "Your signup was successful. Please log in.", "/user/login", http.StatusSeeOther)
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
		app.renderFormErrors(w, r, form, "login.tmpl.html")
		return
	}

	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
			if errors.Is(err, models.ErrInvalidCredentials) {
					form.AddNonFieldError("Email or password incorrect")
					app.renderFormErrors(w, r, form, "login.tmpl.html")
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
	
	// Store the UUID as a string in the session
	app.sessionManager.Put(r.Context(), "authenticatedUserID", id.String())

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	app.setFlashAndRedirect(w, r, "You have been logged out successfully", "/", http.StatusSeeOther)
}