package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pganalyze/pg_query_go/v5"
)

type Schedule interface {
	Next(time.Time) time.Time
}

type Job struct {
	Schedule Schedule
	JobName  string
	DbName   string
	Query    string
	config   *pgx.ConnConfig
	monitor  Monitor
	misc     JobMiscOptions
	valid    bool
}

func CreateJob(jobname, dbname string, s Schedule, target, query string, misc JobMiscOptions, monitor Monitor) (j Job, err error) {
	if jobname == "" || dbname == "" || s == nil {
		return j, fmt.Errorf("Received nil input(s) when creating %s", jobname)
	}
	if query == "" {
		return j, fmt.Errorf("Job %s does not provide a query to run!", jobname)
	}
	if !misc.SkipValidation {
		_, err := pg_query.Parse(query)
		if err != nil {
			return j, fmt.Errorf("Failed to validate query in %s, encountered probable syntax error: %w", jobname, err)
		}
	}

	config, err := pgx.ParseConfig(target)
	if err != nil {
		return j, err
	}
	if config.ConnectTimeout == time.Duration(0) { // Default to 50 seconds if no finite timeout is provided
		config.ConnectTimeout = 50 * time.Second // via the standard pgx & psql PGCONNECT_TIMEOUT env var
	}

	return Job{
		JobName:  jobname,
		DbName:   dbname,
		Schedule: s,
		Query:    query,
		config:   config,
		monitor:  monitor,
		valid:    true,
	}, nil

}

func (j *Job) PrintNextTime(l *log.Logger) {
	l.Printf("%s: %s", j.JobName, j.Schedule.Next(time.Now()))
}

func (j Job) Run() {
	if !j.valid {
		j.monitor.ErrorLog.Printf("ERROR: Invalid pgxcron job %s scheduled!", j.JobName)
		return
	}
	ctx := context.TODO()
	id, terminate := j.monitor.RegisterJob(ctx, j.JobName, j.DbName, j.Query)
	if terminate {
		return
	}
	conn, err := pgx.ConnectConfig(ctx, j.config)
	if err != nil {
		j.monitor.Fail(ctx, id, fmt.Errorf("ERROR: Could not connect to database, aborting %s due to: %w", j.JobName, err))
		return
	}
	defer conn.Close(ctx)

	j.monitor.Run(ctx, id)
	_, err = conn.Exec(ctx, j.Query)
	if err != nil {
		j.monitor.Fail(ctx, id, fmt.Errorf("ERROR: while running %s, failed due to: %w", j.JobName, err))
		return
	}
	j.monitor.Complete(ctx, id)
}

func sortJobsLex(jobs []Job) {
	cmp := func(job1, job2 Job) int {
		if job1.DbName > job2.DbName {
			return 1
		}
		if job1.DbName < job2.DbName {
			return -1
		}
		return strings.Compare(job1.JobName, job2.JobName)
	}
	slices.SortFunc(jobs, cmp)
}
