package main

import (
	"crypto/tls"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"kdg/be/lab/internal/model"
	"kdg/be/lab/internal/models"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
  "github.com/go-playground/form/v4" 
)

type application struct {
  errorLog *log.Logger
  infoLog *log.Logger
  models *model.Models
  geoData *models.GeoData
  users *models.UserModel
  templateCache map[string]*template.Template
  formDecoder *form.Decoder
  sessionManager *scs.SessionManager
}

func main() {
  addr := flag.String("addr", ":4000", "HTTP network address")
  ollama := flag.String("ollama", "llama3", "Ollama model to use")
	flag.Parse()

  infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

  templateCache, err := newTemplateCache()
  if err != nil {
    errorLog.Fatal(err)
  }

  formDecoder := form.NewDecoder()

  sessionManager := scs.New()
  sessionManager.Store = memstore.New()
  sessionManager.Lifetime = 12 * time.Hour
  sessionManager.Cookie.Secure = true

  app := &application{
    errorLog: errorLog,
    infoLog: infoLog,
    models: &model.Models{Model: *ollama},
    geoData: &models.GeoData{} ,
    templateCache: templateCache,
    formDecoder: formDecoder,
    sessionManager: sessionManager,
  }

  tlsConfig := &tls.Config{
    CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
  }

  srv := &http.Server{
    Addr: *addr,
    ErrorLog: errorLog,
    Handler: app.routes(),
    TLSConfig: tlsConfig,
    IdleTimeout: time.Minute,
    ReadTimeout: 5 * time.Second,
    WriteTimeout: 10 * time.Second,
  }

	infoLog.Printf("Starting server on %s", *addr)
  infoLog.Printf("Using ollama model: %s", *ollama)
  err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
  errorLog.Fatal(err)
}
