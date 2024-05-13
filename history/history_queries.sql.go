// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: history_queries.sql

package history

import (
	"context"
)

const createJobRun = `-- name: CreateJobRun :one
insert into jobruns (jobName, database, query, started)
VALUES (?,?,?,?)
returning id
`

type CreateJobRunParams struct {
	Jobname  string
	Database string
	Query    string
	Started  string
}

func (q *Queries) CreateJobRun(ctx context.Context, arg CreateJobRunParams) (int64, error) {
	row := q.queryRow(ctx, q.createJobRunStmt, createJobRun,
		arg.Jobname,
		arg.Database,
		arg.Query,
		arg.Started,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const getRecentRuns = `-- name: GetRecentRuns :many
SELECT id, jobname, jobnumber, "database", "query", started, status from jobruns
where jobName = ?
ORDER BY jobnumber desc
LIMIT 15
`

func (q *Queries) GetRecentRuns(ctx context.Context, jobname string) ([]Jobrun, error) {
	rows, err := q.query(ctx, q.getRecentRunsStmt, getRecentRuns, jobname)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Jobrun
	for rows.Next() {
		var i Jobrun
		if err := rows.Scan(
			&i.ID,
			&i.Jobname,
			&i.Jobnumber,
			&i.Database,
			&i.Query,
			&i.Started,
			&i.Status,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const isDatabaseOnFire = `-- name: IsDatabaseOnFire :one
select exists (select 1 from last_db_status WHERE database = ? AND available = 0)
`

func (q *Queries) IsDatabaseOnFire(ctx context.Context, database string) (int64, error) {
	row := q.queryRow(ctx, q.isDatabaseOnFireStmt, isDatabaseOnFire, database)
	var column_1 int64
	err := row.Scan(&column_1)
	return column_1, err
}

const setDatabaseStatus = `-- name: SetDatabaseStatus :exec
INSERT INTO last_db_status (database, last_seen, available)
VALUES (?, ?, ?)
ON CONFLICT(database) DO UPDATE
SET last_seen = excluded.last_seen, available= excluded.available
`

type SetDatabaseStatusParams struct {
	Database  string
	LastSeen  string
	Available int64
}

func (q *Queries) SetDatabaseStatus(ctx context.Context, arg SetDatabaseStatusParams) error {
	_, err := q.exec(ctx, q.setDatabaseStatusStmt, setDatabaseStatus, arg.Database, arg.LastSeen, arg.Available)
	return err
}

const setJobStatus = `-- name: SetJobStatus :exec
update jobruns
SET status= ?
where id = ?
`

type SetJobStatusParams struct {
	Status string
	ID     int64
}

func (q *Queries) SetJobStatus(ctx context.Context, arg SetJobStatusParams) error {
	_, err := q.exec(ctx, q.setJobStatusStmt, setJobStatus, arg.Status, arg.ID)
	return err
}