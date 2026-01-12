package main

import "os"

type config struct {
	Addr  string
	PgURL string
}

func loadConfig() config {
	addr :=os.Getenv("ADDR")
	if addr=="" {
		addr=":8080"
	}
	pgURL:=os.Getenv("DATABASE_URL")
	if pgURL=="" {
		pgURL="postgres://postgres:postgres@localhost:5432/eon?sslmode=disable"
	}
	return config{Addr: addr,PgURL: pgURL}
}
