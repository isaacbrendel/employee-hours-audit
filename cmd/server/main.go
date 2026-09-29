package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/isaacbrendel/employee-hours-audit/internal/api"
)

func main() {
	static := uiDir("web/dist")
	server := &http.Server{
		Addr:              listenAddr(),
		Handler:           (&api.Server{Static: static}).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	log.Printf("listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func listenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		return ":8080"
	}
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}

func uiDir(path string) fs.FS {
	if _, err := os.Stat(path + "/index.html"); err != nil {
		return nil
	}
	return os.DirFS(path)
}
