
-- name: CreateJobRun :one
insert into jobruns (jobName, database, query, started)
VALUES (?,?,?,?)
returning id;

-- name: SetJobStatus :exec
update jobruns
SET status= ?
where id = ?;

-- name: SetDatabaseStatus :exec
INSERT INTO last_db_status (database, last_seen, available)
VALUES (?, ?, ?)
ON CONFLICT(database) DO UPDATE
SET last_seen = excluded.last_seen, available= excluded.available;


-- name: GetRecentRuns :many
SELECT * from jobruns
where jobName = ?
ORDER BY jobnumber desc
LIMIT 15;

-- name: IsDatabaseOnFire :one
select exists (select 1 from last_db_status WHERE database = ? AND available = 0);