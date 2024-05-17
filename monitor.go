package main

import (
	"context"
	"log"
	"pgxcron/history"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type Monitor struct {
	q          *history.Queries
	ErrorLog   *log.Logger
	ActiveJobs *prometheus.GaugeVec
}

func NewMonitor(db history.DBTX, logger *log.Logger) (m Monitor, err error) {
	queries, err := history.Prepare(context.TODO(), db)
	if err != nil {
		return m, err
	}

	return Monitor{
		q:        queries,
		ErrorLog: logger,
		ActiveJobs: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "active_cron_jobs",
			Help: "number of running cron jobs",
		},
			[]string{"database", "jobname"}),
	}, nil
}

type JobId struct {
	id       int64
	jobname  string
	database string
}

func (m Monitor) RegisterJob(ctx context.Context, jobname, database, query string) (JobId, bool) {
	now := time.Now()
	id, err := m.q.CreateJobRun(ctx, history.CreateJobRunParams{
		Jobname:  jobname,
		Database: database,
		Query:    query,
		Started:  now.Format("2006-01-02 15:04:05 -0700 MST"),
	})
	if err != nil {
		m.ErrorLog.Printf("ERROR: On startup of %s, encountered %s", jobname, err)
		return JobId{}, true
	}
	gauge, err := m.ActiveJobs.GetMetricWithLabelValues(database, jobname)
	if err != nil {
		m.ErrorLog.Printf("ERROR: On startup of %s, had issues finding metric: %s", jobname, err)
		return JobId{}, true
	}
	gauge.Inc()
	return JobId{id: id, jobname: jobname, database: database}, false
}

func (m Monitor) SetStatus(ctx context.Context, id JobId, status string) error {
	return m.q.SetJobStatus(ctx, history.SetJobStatusParams{
		ID:     id.id,
		Status: status,
	})
}

func (m Monitor) Fail(ctx context.Context, id JobId, err error) {
	m.SetStatus(ctx, id, "failed")
	m.ErrorLog.Println(err)
	gauge, err := m.ActiveJobs.GetMetricWithLabelValues(id.database, id.jobname)
	if err != nil {
		m.ErrorLog.Printf("While failing, failed to find metric for failing job: %s", err)
		return
	}
	gauge.Dec()
}

func (m Monitor) Run(ctx context.Context, id JobId) {
	m.SetStatus(ctx, id, "running")
}

func (m Monitor) Complete(ctx context.Context, id JobId) {
	m.SetStatus(ctx, id, "completed")
	gauge, err := m.ActiveJobs.GetMetricWithLabelValues(id.database, id.jobname)
	if err != nil {
		m.ErrorLog.Printf("While failing, failed to find metric for failing job: %s", err)
		return
	}
	gauge.Dec()
}

func (m Monitor) CheckIfOnFire(ctx context.Context, db string) (string, error) {
	boolint, err := m.q.IsDatabaseOnFire(ctx, db)
	if err != nil || boolint == int64(0) {
		return "", err
	}
	return "fire", nil
}

func (m Monitor) JobRunningCount(database, jobname string) (int, error) {
	gauge, err := m.ActiveJobs.GetMetricWithLabelValues(database, jobname)
	if err != nil {
		return 0, err
	}
	metric := &dto.Metric{}
	if err := gauge.Write(metric); err != nil {
		return 0, err
	}
	return int(metric.Gauge.GetValue()), nil
}

// The monitor is also a prometheus collector:
func (m Monitor) Describe(ch chan<- *prometheus.Desc) {
	m.ActiveJobs.Describe(ch)
}

func (m Monitor) Collect(ch chan<- prometheus.Metric) {
	m.ActiveJobs.Collect(ch)
}
