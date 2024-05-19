CREATE TABLE IF NOT EXISTS jobruns (
  id INTEGER PRIMARY KEY NOT NULL,
  jobName text NOT NULL,
  jobnumber integer NOT NULL DEFAULT 0,
  database text NOT NULL,
  query text NOT NULL,
  started text NOT NULL,
  ended text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'connecting'
) STRICT;

CREATE UNIQUE INDEX IF NOT EXISTS seckey on jobruns (jobName, jobnumber);

CREATE TABLE IF NOT EXISTS last_db_status (
  database text PRIMARY KEY NOT NULL,
  last_seen text NOT NULL,
  available integer NOT NULL
) STRICT;


CREATE TABLE IF NOT EXISTS last_finished_job_status (
  jobname text PRIMARY KEY NOT NULL,
  jobnumber integer NOT NULL,
  last_seen text NOT NULL,
  succeeded integer NOT NULL
) STRICT;

CREATE TRIGGER IF NOT EXISTS sequential_jobnumber AFTER INSERT ON jobruns
  BEGIN
    UPDATE jobruns
    SET jobnumber = (select max(jobnumber)+1 from jobruns where jobName = NEW.jobName)
    WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS last_seen_update AFTER UPDATE OF status ON jobruns
WHEN NEW.status in ('failed','completed')
BEGIN
  INSERT INTO last_db_status (database, last_seen, available)
  VALUES (NEW.database, NEW.started, NEW.status <> 'failed') 
  ON CONFLICT(database) DO UPDATE
  SET last_seen = excluded.last_seen, available= excluded.available;
  INSERT INTO last_finished_job_status (jobname, jobnumber, last_seen, succeeded)
  VALUES (NEW.jobName, NEW.jobnumber ,NEW.started, NEW.status <> 'failed')
  ON CONFLICT(jobname) DO UPDATE
  SET jobnumber=excluded.jobnumber, last_seen = excluded.last_seen, succeeded= excluded.succeeded;
END;

CREATE TRIGGER IF NOT EXISTS clean_old_runs
AFTER UPDATE OF jobnumber ON last_finished_job_status
BEGIN
    DELETE FROM jobruns WHERE jobName = NEW.jobname AND jobnumber < NEW.jobnumber - 1000;
END;