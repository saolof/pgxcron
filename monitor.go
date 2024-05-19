package main

import (
	"context"
	"github.com/saolof/pgxcron/history"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type Monitor struct {
	q                  *history.Queries
	ErrorLog           *log.Logger
	ActiveJobs         *prometheus.GaugeVec
	MetricDescriptions map[string]*prometheus.Desc
}

func NewMonitor(db history.DBTX, logger *log.Logger) (m Monitor, err error) {
	queries, err := history.Prepare(context.TODO(), db)
	if err != nil {
		return m, err
	}

	db_status := prometheus.NewDesc("database_status", "Returns 1 if the database is available", []string{"database"}, map[string]string{})
	job_status := prometheus.NewDesc("last_job_status", "Returns 1 if the lastest finished job succeeded", []string{"jobname"}, map[string]string{})
	descriptions := map[string]*prometheus.Desc{"database_status": db_status, "last_job_status": job_status}

	return Monitor{
		q:        queries,
		ErrorLog: logger,
		ActiveJobs: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "active_cron_jobs",
			Help: "number of running cron jobs",
		},
			[]string{"database", "jobname"}),
		MetricDescriptions: descriptions,
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

func (m Monitor) Run(ctx context.Context, id JobId) {
	err := m.q.SetJobStatus(ctx, history.SetJobStatusParams{
		ID:     id.id,
		Status: "running",
	})
	if err != nil {
		m.ErrorLog.Printf("Error writing status update to sqlite: %s", err)
	}
}

func (m Monitor) endJob(ctx context.Context, id JobId, status string) error {
	return m.q.MakJobAsFinished(ctx, history.MakJobAsFinishedParams{
		ID:     id.id,
		Status: status,
		Ended:  time.Now().Format("2006-01-02 15:04:05 -0700 MST"),
	})
}

func (m Monitor) Fail(ctx context.Context, id JobId, err error) {
	if err := m.endJob(ctx, id, "failed"); err != nil {
		m.ErrorLog.Printf("Error writing status update to sqlite: %s", err)
	}
	m.ErrorLog.Println(err)
	gauge, err := m.ActiveJobs.GetMetricWithLabelValues(id.database, id.jobname)
	if err != nil {
		m.ErrorLog.Printf("While failing, failed to find metric for failing job: %s", err)
		return
	}
	gauge.Dec()
}

func (m Monitor) Complete(ctx context.Context, id JobId) {
	if err := m.endJob(ctx, id, "completed"); err != nil {
		m.ErrorLog.Printf("Error writing status update to sqlite: %s", err)
	}
	gauge, err := m.ActiveJobs.GetMetricWithLabelValues(id.database, id.jobname)
	if err != nil {
		m.ErrorLog.Printf("While failing, failed to find metric for failing job: %s", err)
		return
	}
	gauge.Dec()
}

func (m Monitor) OnFireStatus(ctx context.Context) (map[string]bool, error) {
	statuses := map[string]bool{}
	statvec, err := m.q.LastDatabaseStatus(ctx)
	if err != nil {
		return statuses, err // Return empty map instead of nil
	}
	for _, val := range statvec {
		statuses[val.Database] = val.Onfire
	}
	return statuses, nil
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

// Implements the prometheus Collector interface
func (m Monitor) Describe(ch chan<- *prometheus.Desc) {
	m.ActiveJobs.Describe(ch)
	for _, desc := range m.MetricDescriptions {
		ch <- desc
	}
}

func (m Monitor) collectDatabaseStatuses(desc *prometheus.Desc, ch chan<- prometheus.Metric) {
	onfiremap, err := m.OnFireStatus(context.TODO())
	if err != nil {
		return
	}
	for database, isonfire := range onfiremap {
		onfire := 1.0
		if isonfire {
			onfire = 0.0
		}
		metric, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, onfire, database)
		if err != nil {
			metric = prometheus.NewInvalidMetric(desc, err)
		}
		ch <- metric
	}
}

func (m Monitor) collectJobStatuses(desc *prometheus.Desc, ch chan<- prometheus.Metric) {
	statuses, err := m.q.LastJobCompletedStatus(context.TODO())
	if err != nil {
		return
	}
	for _, status := range statuses {
		metric, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, float64(status.Succeeded), status.Jobname)
		if err != nil {
			metric = prometheus.NewInvalidMetric(desc, err)
		}
		ch <- metric
	}
}

// Implements the prometheus Collector interface
func (m Monitor) Collect(ch chan<- prometheus.Metric) {
	m.ActiveJobs.Collect(ch)
	if desc, ok := m.MetricDescriptions["database_status"]; ok {
		m.collectDatabaseStatuses(desc, ch)
	}
	if desc, ok := m.MetricDescriptions["last_job_status"]; ok {
		m.collectJobStatuses(desc, ch)
	}
}
