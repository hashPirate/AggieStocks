// Ingress service of my HTTP API that accepts events and appends to the event log.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"eon/platform/internal/ingress"
	 elog "eon/platform/internal/log"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg :=loadConfig()
	ctx :=context.Background()
	pool,err :=pgxpool.New(ctx,cfg.PgURL)
	if err!=nil {
		log.Fatal("pg pool:",err)
	}
	defer pool.Close()
	eventLog:=elog.NewPostgresLog(pool,"event_log")
	if err :=eventLog.EnsureSchema(ctx); err!=nil {
		log.Fatal("event_log schema:",err)
	}
	handler :=&ingress.Handler{Log: eventLog}
	mux:=http.NewServeMux()
	mux.Handle("/v1/events",handler)
	mux.HandleFunc("/health",func(w http.ResponseWriter,_ *http.Request) {w.WriteHeader(http.StatusOK) })
	srv :=&http.Server{Addr: cfg.Addr,Handler: mux}
	go func() {
		if err:=srv.ListenAndServe(); err!=nil && err!=http.ErrServerClosed {
			log.Fatal("server:",err)
		}
	}()
	quit :=make(chan os.Signal,1)
	signal.Notify(quit,syscall.SIGINT,syscall.SIGTERM)
	<-quit
	if err :=srv.Shutdown(ctx); err!=nil {
		log.Print("shutdown:",err)
	}
	_ =eventLog.Close()
}
