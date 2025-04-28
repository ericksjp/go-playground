package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	// server instance
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		// the http server will use our jsonlogger to log the messages.
		// he can be passed as argument here because implements the io.Writer
		// interface
		ErrorLog:     log.New(app.logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// comunicates with the "shutdown" routine
	shutdownError := make(chan error)

	// go routine that will listen for incoming signals and try to gracefully stop the server
	go func() {
		// for some reason, the compiler complains if the channel is unbufered
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// waiting for a signal
		s := <- quit

		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		// throw error if it takes more than 5 seconds to shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// writing to the channel only if the server is not stopped gracefully
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		// block until the counter is 0
		app.wg.Wait()
		shutdownError <- nil
	}()

	// passing a map containing additinal properties
	app.logger.PrintInfo("starting server", map[string]string{
		"env": app.config.env,
		"addr": srv.Addr,
	})


	// return error if its not the expected for when the server is stopped
	// gracefully
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// return error if its not nil (server stopped gracefully)
	err = <- shutdownError
	if err != nil {
		return err
	}

	// no errors occured
	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
