
# PGXCron

![Pgxcron logo, elephant with a clock on its head](/logo.webp)


pgxcron is a service written in Go (so it is a self-contained portable binary) that reads a simple toml-formatted crontab file:

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

...and once started up with those configuration files it should just execute the queries on a schedule.
The daemon is stateless (apart from the monitoring HTTP endpoint), so its behaviour is only defined by the
config file at startup and there should not be any consequences to restarting it.

The cron library used internally is the same one as the one used by kubernetes. Standard POSIX cron syntax is used. 
It also embeds the postgres parser and will syntax check all queries on startup, and will refuse to start if any query
contains a syntax error.


## Monitoring dashboard & prometheus integration

Apart from logging to stdout, pgxcron also provides an optional HTTP server. The /jobs endpoint serves you an html
dashboard with the jobs at a glance which can be used to browse job history. The /metrics endpoint exposes a set of prometheus metrics.

This is intended to make it useful for cases where jobs may be run on many different database instances while still
providing an easy overview of what jobs succeeded and failed, and makes it viable even for status checks.

The HTTP endpoints are and will always remain just views into what is happening for monitoring purposes,
and cannot change behaviour. Any behaviour changes intentionally have to be deployed by changing the config file.

## Comparison to other tools

Pgxcron is intended to address a corner of the design space not addressed by standard crontab or by extensions such as pg_cron 
or the timescaledb scheduler, and primarily values ease of use & maintenance and a declarative configuration file format to 
avoid configuration drift and facilitate version control.

If you want to do something that has significant side effects, use some variation of the standard cron running a shell script or other command.
If you want to perform tasks closely coupled to a given schema such as reliably creating a new partition of a table once per month,
use pg_cron and put the command that creates the job in a migration along with other DDL statements.

If you just want an easy way to run simple queries on a large number of databases that does not perform things like schema changes,
and does not have other side effects, then pgxcron is intended to fill that gap.


## Near term priorities

This is somewhat usable but in a somewhat early state. Near term priorities include improving the logging using slog,
adding a few more configuration options with sane defaults, implementing an automated integration test workflow,
and packaging for alpine apk.
