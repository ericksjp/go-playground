package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env string
}

type application struct {
	config config
	logger *log.Logger
}

// healthCheckHandler responds to /v1/healthcheck with a JSON object
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request)  {
	data := envelope{
		"status": "available",
		"system_info": map[string]string{
			"version": version,
			"env": app.config.env,
		},
	}
	app.writeJSON(w, 200, data, nil)
}

func main() {

	// gettin cmd args
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "api port")
	flag.StringVar(&cfg.env, "env", "development", "api env (development|staging|production)")
	flag.Parse()

	// server logger
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	app := &application{
		config: cfg,
		logger: logger,
	}

	// server instance
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)
	err := srv.ListenAndServe()
	logger.Fatal(err)
}
