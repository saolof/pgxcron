package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/robfig/cron/v3"
)

type DatabaseConfig struct {
	ConnString    string
	PasswordVar   string
	JustUsePgPass bool
}

func DecodeDatabases(crontab io.Reader, usepgpass bool) (map[string]string, error) {
	var configs map[string]DatabaseConfig
	decoder := toml.NewDecoder(crontab)
	err := decoder.Decode(&configs)
	if err != nil {
		return nil, err
	}
	databases := map[string]string{}
	for key, config := range configs {
		if config.ConnString == "" {
			return nil, fmt.Errorf("Missing connstring in database %s", key)
		}
		if usepgpass || config.JustUsePgPass {
			databases[key] = strings.Replace(config.ConnString, ":$password", "", 1)
		} else if config.PasswordVar == "" {
			databases[key] = config.ConnString
		} else {
			password := os.Getenv(config.PasswordVar)
			if password == "" {
				return nil, fmt.Errorf("Injected passwordvar %s is empty!", config.PasswordVar)
			}
			databases[key] = strings.Replace(config.ConnString, "$password", password, 1)
		}
	}
	return databases, nil
}

type JobMiscOptions struct {
	SkipValidation      bool
	AllowConcurrentJobs bool
}

type JobConfig struct {
	CronSchedule string
	Database     string
	Query        string
	JobMiscOptions
}

func DecodeJobs(crontab io.Reader) (jobconfigs map[string]JobConfig, err error) {
	decoder := toml.NewDecoder(crontab)
	err = decoder.Decode(&jobconfigs)
	if err != nil {
		return nil, err
	}
	return jobconfigs, nil
}

func CreateJobs(configs map[string]JobConfig, databases map[string]string, monitor Monitor) ([]Job, error) {
	jobs := []Job{}
	for name, config := range configs {
		schedule, err := cron.ParseStandard(config.CronSchedule)
		if err != nil {
			return nil, fmt.Errorf("Cron schedule error: %w", err)
		}
		connstr, ok := databases[config.Database]
		if !ok {
			return nil, fmt.Errorf("Missing Db: The database %s specified by job %s does not seem to exist!", config.Database, name)
		}
		job, err := CreateJob(name, config.Database, schedule, connstr, config.Query, config.JobMiscOptions, monitor)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	sortJobsLex(jobs) // since iterating over map keys is random.
	return jobs, nil
}
