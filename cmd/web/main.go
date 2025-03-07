package main

import (
	"crypto/tls"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"kdg/be/lab/internal/db"
	"kdg/be/lab/internal/model"
	"kdg/be/lab/internal/models"

	"github.com/BurntSushi/toml"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	models         *model.Models
	chatPort       *model.ChatPort
	geoData        *models.GeoData
	users          *models.UserModel
	chats          *models.ChatModel
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
	i18nBundle     *i18n.Bundle
}

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
	ollama := flag.String("ollama", "llama3", "Ollama model to use")
	dsn := flag.String("dsn", "data/sqlite_lab.db", "sqlite data source name")
	chatPort := flag.String("chatPort", ":8000", "Chat server network address")
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	db, err := db.OpenDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	i18nBundle, err := init18n()
	if err != nil {
		errorLog.Fatal(err)
	}

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.Cookie.Secure = true

	app := &application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		models:         &model.Models{Model: *ollama},
		chatPort:       &model.ChatPort{Port: *chatPort},
		geoData:        &models.GeoData{},
		users:          &models.UserModel{DB: db},
		chats:          &models.ChatModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
		i18nBundle:     i18nBundle,
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      app.routes(),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("Starting server on %s", *addr)
	infoLog.Printf("Using ollama model: %s", *ollama)
	infoLog.Printf("Using sqlite database: %s", *dsn)
	infoLog.Printf("Starting chat server on %s", *chatPort)
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}

func init18n() (*i18n.Bundle, error) {
	bundle := i18n.NewBundle(language.English)

	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	if _, err := bundle.LoadMessageFile("locales/active.en.toml"); err != nil {
		return nil, err
	}
	if _, err := bundle.LoadMessageFile("locales/active.nl.toml"); err != nil {
		return nil, err
	}
	return bundle, nil
}
