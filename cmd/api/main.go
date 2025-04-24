package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"


	// for some reason the pq driver needs to be registered with the sql
	// package in order to work. so it has a init function only for this
	// purpose and we need to activate it
	_ "github.com/lib/pq"

	"github.com/ericksjp703/greenlight/internal/data"
)

const version = "1.0.0"

type config struct {
	port int
	env string
	db struct {
		dns string
	}
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

	// initialize the singletown instance
	data.GetMoviesStore()

	// gettin cmd args
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "api port")
	flag.StringVar(&cfg.env, "env", "development", "api env (development|staging|production)")
	flag.StringVar(&cfg.db.dns, "db-dns", "postgres://postgres:postgres@localhost/postgres", "postgresql dns")
	flag.Parse()

	// server logger
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	// open a connection pool to the database, exits if error
	db, err := openDB(&cfg)
	if err != nil {
		logger.Fatal(err)
	}
	// ensure the connection pool is closed before the main function exits
	defer db.Close()
	logger.Println("database connection established")

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
	err = srv.ListenAndServe()
	logger.Fatal(err)
}

// creates a connection pool and verifies if everything is ok
func openDB(cfg *config) (*sql.DB, error) {
	// creates a empty connection pool using the dns string from config
	db, err := sql.Open("postgres", cfg.db.dns + "?sslmode=disable")
	if err != nil {
		return nil, err
	}

	// Creates a context with a 5-second timeout, ensuring PingContext cancels
	// if the connection is not established within that time
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	// establish a connection to the database using the context
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	
	return db, nil
}
