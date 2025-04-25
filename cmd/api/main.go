package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	env  string
	db   struct {
		dns             string
		maxOpenConns    int
		maxIdleConns    int
		maxIdleTime     string
		maxLifetime 	string
	}
}

type application struct {
	config config
	logger *log.Logger
	models data.Models
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

func getenvAsInt(key string) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("the environment variable '%s' should be specified", key)
	}

	intVal, err := strconv.Atoi(value);
	if err != nil {
		log.Fatalf("the environment variable '%s' should be an integer", key)
	}

	return intVal
}

func main() {

	// gettin cmd args
	var cfg config

	// Load command-line flags with environment variable defaults
	flag.IntVar(&cfg.port, "port", getenvAsInt("PORT"), "api port")
	flag.StringVar(&cfg.env, "env", os.Getenv("ENV"), "api env (development|staging|production)")

	// Database configuration
	flag.StringVar(&cfg.db.dns, "db-dns", os.Getenv("DB_DNS"), "postgresql dns")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", getenvAsInt("DB_MAX_OPEN_CONNS"), "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", getenvAsInt("DB_MAX_IDLE_CONNS"), "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", os.Getenv("DB_MAX_IDLE_TIME"), "PostgreSQL max connection idle time")
	flag.StringVar(&cfg.db.maxLifetime, "db-max-lifetime", os.Getenv("DB_MAX_LIFETIME"), "PostgreSQL max connection lifetime")

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
		models: data.NewModels(db),
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

	// set the maximum number of open connections (both idle and in-use) allowed in the connection pool
	db.SetMaxOpenConns(cfg.db.maxOpenConns);
	// set the maximum number of idle connections allowed in the connection pool
	db.SetMaxIdleConns(cfg.db.maxIdleConns);
	// set the maximum duration a connection can remain idle in the pool before being closed
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration);
	// set the maximum lifetime of a connection in the pool, after which it will be closed and removed
	duration, err = time.ParseDuration(cfg.db.maxLifetime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(duration);

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
