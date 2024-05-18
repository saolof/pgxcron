
# PGXCron

![Pgxcron logo, elephant with a clock on its head](/logo.webp)


pgxcron is a service that ships as a portable, self-contained binary, that reads a simple toml-formatted crontab file:

```toml
[example-healthcheck]
cronschedule="* * * * *"
database="example"
query="select 1"

[example-analyze]
cronschedule="0 6 * * *"
database="example"
query="analyze"

[test-healthcheck]
cronschedule="* * * * *"
database="test"
query="select 1"

[test-analyze]
cronschedule="0 */4 * * *"
database="test"
query="analyze"
```

And a toml file with your databases in this format:

```toml
[example]
# Secret injection example, you can name a specific env var to replace $password in a connstring
connstring="postgresql://admin:$password@exampleurl.net:5432/postgres"
passwordvar="EXAMPLE_PASS"

[test]
# Just relies on pgpass or standard psql env vars
connstring="postgresql://admin@localhost:5432/postgres"
```

...and once started, will execute the queries on the schedule in the config. POSIX cron schedule syntax is used.
The daemon is stateless (apart from the monitoring HTTP endpoint), so its behaviour is only defined by the
config file at startup and is deterministic.

It also extensively lints its config file, both the cron syntax and the query (using a statically linked copy of the postgres parser).

## Monitoring dashboard and Prometheus integration

Apart from logging to stdout, pgxcron also provides an optional HTTP server. 
The /jobs endpoint serves you an HTML dashboard with the jobs at a glance, which can be used to browse job history. 
The /metrics endpoint exposes a set of prometheus metrics, allowing it to double as a "poor man's postgres_exporter" for monitoring purposes.

The HTTP endpoints are and will always remain just views into what is happening for monitoring purposes,
and cannot change behaviour. Any behaviour changes intentionally have to be deployed by changing the config file.

## Comparison to other tools

Pgxcron is intended to address a corner of the design space not addressed by standard crontab or by extensions such as pg_cron 
or the timescaledb scheduler, and primarily values ease of use when managing many databases, avoiding configuration drift, and facilitating version control.

If you want to do something that has significant side effects, use some variation of the standard cron running a shell script or other command.
If you want to perform tasks closely coupled to a given schema, such as reliably creating a new partition of a table once per month,
use pg_cron and put the command that creates the job in a migration along with other DDL statements.

If you just want an easy way to run simple queries on a large number of databases that does not perform things like schema changes,
and does not have other side effects, then pgxcron is intended to fill that gap.

## Near term priorities

This is approaching the point where the core functionality has fairly minimal maintenance needs beyond updating dependencies.

Todo list:
* Setting up a github actions workflow for automated continuous integration tests in the github repository, and versioning/release.
* Improving the logging format. robfig/cron uses a logr-subset logger interface, which slightly differs from slog.
* Finalizing the non-core configuration options and their defaults.
* Packaging for Alpine and Debian
