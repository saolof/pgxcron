package main

import (
	"context"
	"log"
	"pgxcron/history"
	"time"
)

type Monitor struct {
	q        *history.Queries
	ErrorLog *log.Logger
}

func NewMonitor(db history.DBTX, logger *log.Logger) (m Monitor, err error) {
	queries, err := history.Prepare(context.TODO(), db)
	if err != nil {
		return m, err
	}

	return Monitor{
		q:        queries,
		ErrorLog: logger,
	}, nil
}

func (m Monitor) RegisterJob(ctx context.Context, jobname, database, query string) (id int64, terminate bool) {
	id, err := m.q.CreateJobRun(ctx, history.CreateJobRunParams{
		Jobname:  jobname,
		Database: database,
		Query:    query,
		Started:  time.Now().Format("2006-01-02 15:04:05 -0700 MST"),
	})
	if err != nil {
		m.ErrorLog.Printf("ERROR: On startup of %s, encountered %s", jobname, err)
		return id, true
	}
	return id, false
}

func (m Monitor) SetStatus(ctx context.Context, id int64, status string) error {
	return m.q.SetJobStatus(ctx, history.SetJobStatusParams{
		ID:     id,
		Status: status,
	})
}

func (m Monitor) Fail(ctx context.Context, id int64, err error) {
	m.SetStatus(ctx, id, "failed")
	m.ErrorLog.Println(err)
}

func (m Monitor) Run(ctx context.Context, id int64) {
	m.SetStatus(ctx, id, "running")
}

func (m Monitor) Complete(ctx context.Context, id int64) {
	m.SetStatus(ctx, id, "completed")
}

func (m Monitor) CheckIfOnFire(ctx context.Context, db string) (string, error) {
	boolint, err := m.q.IsDatabaseOnFire(ctx, db)
	if err != nil || boolint == int64(0) {
		return "", err
	}
	return "fire", nil
}
