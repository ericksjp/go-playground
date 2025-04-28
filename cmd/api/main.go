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
	"github.com/ericksjp703/greenlight/internal/jsonlog"
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

	limiter struct {
		rps float64
		burst int
		enabled bool
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
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

func getenvAsBool(key string) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("the environment variable '%s' should be specified", key)
	}

	boolVal, err := strconv.ParseBool(value);
	if err != nil {
		log.Fatalf("the environment variable '%s' should be a boolean", key)
	}

	return boolVal
}

func getEnvAsFloat(key string) float64 {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("the environment variable '%s' should be specified", key)
	}

	floatVal, err := strconv.ParseFloat(value, 64);
	if err != nil {
		log.Fatalf("the environment variable '%s' should be a float", key)
	}

	return floatVal
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

	// Rate limiter configuration
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", getEnvAsFloat("LIMITER_RPS"), "Rate limiter requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", getenvAsInt("LIMITER_BURST"), "Rate limiter burst capacity")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", getenvAsBool("LIMITER_ENABLED"), "Enable or disable the rate limiter")

	flag.Parse()

	// write to stdout and Logs >= LevelInfo
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// open a connection pool to the database, exits if error
	db, err := openDB(&cfg)
	if err != nil {
		// using the logger to print the error message and exit the program
		logger.PrintFatal(err, nil)
	}
	// ensure the connection pool is closed before the main function exits
	defer db.Close()

	logger.PrintInfo("database connection established", nil)

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	// server instance
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		// the http server will use our jsonlogger to log the messages.
		// he can be passed as argument here because implements the io.Writer
		// interface
		ErrorLog:     log.New(logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// passing a map containing additinal properties
	logger.PrintInfo("starting server", map[string]string{
		"env": cfg.env,
		"addr": srv.Addr,
	})

	err = srv.ListenAndServe()
	// using the fatal print to log the error and quit the process
	logger.PrintFatal(err, nil)
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
