package main

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
	"io"
	"log"
	"os"
)

//go:embed history_schema.sql
var ddl string

func makeJobs(cronfile string, databasefile string, logger *log.Logger, monitor Monitor, usepgpass bool) ([]Job, error) {
	crontab, err := os.Open(cronfile)
	if err != nil {
		return nil, fmt.Errorf("Error opening cronfile: %w", err)
	}
	defer crontab.Close()
	dbtoml, err := os.Open(databasefile)
	if err != nil {
		return nil, fmt.Errorf("Error opening dbfile: %w", err)
	}
	defer dbtoml.Close()
	databases, err := DecodeDatabases(dbtoml, usepgpass)
	if err != nil {
		return nil, fmt.Errorf("Error reading db file: %w", err)
	}
	jobconfigs, err := DecodeJobs(crontab)
	if err != nil {
		return nil, fmt.Errorf("Error reading crontab file: %w", err)
	}
	return CreateJobs(jobconfigs, databases, monitor)
}

func run(ctx context.Context, w io.Writer, logger *log.Logger, webport int, check bool, crontab, databases, historyfile string, args []string) error {
	db, err := sql.Open("sqlite3", historyfile)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1) // Only seems to be necessary for in-memory Db
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("Error creating tables: %w", err)
	}

	monitor, err := NewMonitor(db, logger)
	if err != nil {
		return fmt.Errorf("Error setting up monitoring: %w", err)
	}
	jobs, err := makeJobs(crontab, databases, logger, monitor, check)
	if err != nil {
		return err
	}
	if check {
		fmt.Println("Validated config files.")
		return nil
	}
	fmt.Println("Validated config files, starting up cron jobs...")
	c := cron.New()
	for _, job := range jobs {
		c.Schedule(job.Schedule, job)
	}
	if webport > 0 && webport <= 49152 {
		server := webserver(webport, jobs, monitor)
		go server.ListenAndServe()
		fmt.Println("Listening to traffic on port ", webport)
	}
	c.Run()
	return nil
}

func main() {
	databases := "databases.toml"
	crontab := "crontab.toml"
	historyfile := "file::memory:?cache=shared"
	flag.StringVar(&databases, "databases", databases, "Path to the list of databases.")
	flag.StringVar(&crontab, "crontab", crontab, "Path to the list of cron jobs.")
	flag.StringVar(&historyfile, "historyfile", historyfile, "Path to the database file used for job history logging.")
	webport := flag.Int("webport", 8035, "The port used for the web interface that can be used to check on recent jobs. Set to zero to disable the web interface.")
	check := flag.Bool("check", false, "This flag disables spinning up the cron jobs and just syntax checks the config.")
	flag.Parse()
	ctx := context.Background()
	if err := run(ctx, os.Stdout, log.Default(), *webport, *check, crontab, databases, historyfile, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
