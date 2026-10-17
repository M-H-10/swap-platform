// Command migrate applies or rolls back the SQL files in /migrations.
//
// Usage (from the backend-go folder):
//
//	go run ./cmd/migrate up        apply every migration not applied yet
//	go run ./cmd/migrate down      roll back the last migration only
//	go run ./cmd/migrate version   show the current version
//	go run ./cmd/migrate force N   mark version N as clean (repair only)
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the "pgx5" driver
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/M-H-10/swap-platform/backend-go/configs"
	"github.com/M-H-10/swap-platform/backend-go/migrations"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatalf("reading migration files: %v", err)
	}

	// golang-migrate picks the driver from the URL scheme.
	dbURL := strings.Replace(cfg.DatabaseURL, "postgres://", "pgx5://", 1)
	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer m.Close()

	switch os.Args[1] {
	case "up":
		err = m.Up()
	case "down":
		// One step at a time, never everything at once.
		if _, _, vErr := m.Version(); errors.Is(vErr, migrate.ErrNilVersion) {
			err = migrate.ErrNoChange // nothing to roll back
		} else {
			err = m.Steps(-1)
		}
	case "version":
		// nothing to do, the version is printed below
	case "force":
		if len(os.Args) < 3 {
			usage()
		}
		v, convErr := strconv.Atoi(os.Args[2])
		if convErr != nil {
			log.Fatalf("force needs a number, got %q", os.Args[2])
		}
		err = m.Force(v)
	default:
		usage()
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("nothing to do")
	} else if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	version, dirty, err := m.Version()
	switch {
	case errors.Is(err, migrate.ErrNilVersion):
		log.Println("current version: none (empty database)")
	case err != nil:
		log.Fatalf("reading version: %v", err)
	default:
		log.Printf("current version: %d (dirty: %v)", version, dirty)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: go run ./cmd/migrate [up | down | version | force N]")
	os.Exit(2)
}
