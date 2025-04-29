package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	// for some reason the pq driver needs to be registered with the sql
	// package in order to work. so it has a init function only for this
	// purpose and we need to activate it
	_ "github.com/lib/pq"

	"github.com/ericksjp703/greenlight/internal/data"
	"github.com/ericksjp703/greenlight/internal/jsonlog"
	"github.com/ericksjp703/greenlight/internal/mailer"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dns          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
		maxLifetime  string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg sync.WaitGroup
}

// healthCheckHandler responds to /v1/healthcheck with a JSON object
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	data := envelope{
		"status": "available",
		"system_info": map[string]string{
			"version": version,
			"env":     app.config.env,
		},
	}
	app.writeJSON(w, 200, data, nil)
}

func envOrDef[T comparable](key string, defaultValue T) T {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	var err error
	var result any

	switch any(defaultValue).(type) {
	case string:
		result = value
	case int:
		result, err = strconv.Atoi(value)
	case bool:
		result, err = strconv.ParseBool(value)
	case float64:
		result, err = strconv.ParseFloat(value, 64)
	default:
		log.Fatalf("unsupported type for environment variable '%s'", key)
	}

	if err != nil {
		log.Fatalf("invalid type for environment variable '%s'", key)
	}

	return result.(T)
}

func main() {

	// gettin cmd args
	var cfg config

	// Load command-line flags with environment variable defaults
	flag.IntVar(&cfg.port, "port", envOrDef("PORT", 3000), "api port")
	flag.StringVar(&cfg.env, "env", envOrDef("ENV", "development"), "api env (development|staging|production)")

	// Database configuration
	flag.StringVar(&cfg.db.dns, "db-dns", os.Getenv("DB_DNS"), "postgresql dns")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", envOrDef("DB_MAX_OPEN_CONNS", 25), "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", envOrDef("DB_MAX_IDLE_CONNS", 25), "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", envOrDef("DB_MAX_IDLE_TIME", "15m"), "PostgreSQL max connection idle time")
	flag.StringVar(&cfg.db.maxLifetime, "db-max-lifetime", envOrDef("DB_MAX_LIFETIME", "0m"), "PostgreSQL max connection lifetime")

	// Rate limiter configuration
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", envOrDef("LIMITER_RPS", float64(2)), "Rate limiter requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", envOrDef("LIMITER_BURST", 4), "Rate limiter burst capacity")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", envOrDef("LIMITER_ENABLED", true), "Enable or disable the rate limiter")

	// SMTP configuration
	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", envOrDef("SMTP_PORT", 587), "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", os.Getenv("SMTP_SENDER"), "SMTP sender")

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
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	os.Exit(0)
}

// creates a connection pool and verifies if everything is ok
func openDB(cfg *config) (*sql.DB, error) {
	// creates a empty connection pool using the dns string from config
	db, err := sql.Open("postgres", cfg.db.dns+"?sslmode=disable")
	if err != nil {
		return nil, err
	}

	// set the maximum number of open connections (both idle and in-use) allowed in the connection pool
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	// set the maximum number of idle connections allowed in the connection pool
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	// set the maximum duration a connection can remain idle in the pool before being closed
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)
	// set the maximum lifetime of a connection in the pool, after which it will be closed and removed
	duration, err = time.ParseDuration(cfg.db.maxLifetime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(duration)

	// Creates a context with a 5-second timeout, ensuring PingContext cancels
	// if the connection is not established within that time
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// establish a connection to the database using the context
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
