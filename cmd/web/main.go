package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"kdg/be/lab/internal/model"
)

type application struct {
  errorLog *log.Logger
  infoLog *log.Logger
  models *model.Models
}

func main() {
  addr := flag.String("addr", ":4000", "HTTP network address")
  ollama := flag.String("ollama", "llama3", "Ollama model to use")
	flag.Parse()

  infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

  app := &application{
    errorLog: errorLog,
    infoLog: infoLog,
    models: &model.Models{Model: *ollama},
  }

  srv := &http.Server{
    Addr: *addr,
    ErrorLog: errorLog,
    Handler: app.routes(),
  }

	infoLog.Printf("Starting server on %s", *addr)
  infoLog.Printf("Using ollama model: %s", *ollama)
  err := srv.ListenAndServe()
  errorLog.Fatal(err)
}
