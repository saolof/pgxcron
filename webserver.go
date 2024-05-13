package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"pgxcron/history"
	"slices"
	"time"
)

//go:embed static
var FSstatic embed.FS

//go:embed templates/*.html
var FStemplates embed.FS

var templates = template.Must(template.ParseFS(FStemplates, "templates/*.html"))

type DbDisplay struct {
	Database string
	OnFire   string
}

type JobDisplay struct {
	Database         string
	Name             string
	Query            string
	Icon             string
	Nextrun          time.Time
	Runs             []history.Jobrun
	OpenDbTag        bool // To split by databases in a flat for loop
	CloseDbTag       bool
	DatabaseIsOnFire string
}

func computeJobDisplay(ctx context.Context, m Monitor, now time.Time, job Job) (display JobDisplay, err error) {
	recent, err := m.q.GetRecentRuns(ctx, job.JobName) // SQLite is in-memory so O(N) prepared queries is ok
	if err != nil {
		return display, err
	}
	display = JobDisplay{
		Database: job.DbName,
		Name:     job.JobName,
		Query:    job.Query,
		Icon:     "🔵",
		Nextrun:  job.Schedule.Next(now),
		Runs:     recent,
	}
	if len(recent) > 0 && recent[0].Status == "failed" {
		display.Icon = "🔴"
	}
	if len(recent) > 0 && recent[0].Status == "completed" {
		display.Icon = "🟢"
	}
	return
}

func compareJobDisplays(job1, job2 JobDisplay) int {
	if job1.Database > job2.Database {
		return 1
	}
	if job1.Database < job2.Database {
		return -1
	}
	return job1.Nextrun.Compare(job2.Nextrun)
}

func sortJobDisplays(jobs []JobDisplay) {
	slices.SortStableFunc(jobs, compareJobDisplays)
}

func showjobs(jobs []Job, m Monitor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		now := time.Now()
		jobdisplays := make([]JobDisplay, len(jobs))
		for i, job := range jobs {
			jobdisplays[i], err = computeJobDisplay(r.Context(), m, now, job)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		sortJobDisplays(jobdisplays)
		prev_db := ""
		isonfire := ""
		for i := range jobdisplays {
			if jobdisplays[i].Database != prev_db {
				jobdisplays[i].OpenDbTag = true
				isonfire, _ = m.CheckIfOnFire(r.Context(), jobdisplays[i].Database)
				jobdisplays[i].CloseDbTag = i != 0
				prev_db = jobdisplays[i].Database
			}
			jobdisplays[i].DatabaseIsOnFire = isonfire
		}
		if len(jobdisplays) > 0 {
			jobdisplays[len(jobdisplays)-1].CloseDbTag = true
		}

		err = templates.ExecuteTemplate(w, "jobspage", jobdisplays)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

}

func webserver(port int, jobs []Job, monitor Monitor) *http.Server {

	mux := http.NewServeMux()
	mux.Handle("GET /static/", setHeader(http.FileServer(http.FS(FSstatic)), "Cache-Control", "max-age=86400"))
	mux.HandleFunc("GET /jobs", showjobs(jobs, monitor))

	var h http.Handler = mux
	//	h = middleware.Logger(h)
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h,
	}
}

func setHeader(handler http.Handler, header, value string) http.Handler {
	return http.HandlerFunc(func(h http.ResponseWriter, r *http.Request) {
		h.Header().Set(header, value)
		handler.ServeHTTP(h, r)
	})
}
