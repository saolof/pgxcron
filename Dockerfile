from golang:latest as builder

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

WORKDIR /pgxcron
COPY go.mod .
COPY go.sum .
COPY history_schema.sql .
COPY history_queries.sql .
COPY sqlc.yaml .
RUN go mod download
RUN sqlc generate
COPY . .
RUN go build

from busybox:glibc

COPY --from=builder /pgxcron/pgxcron /bin/pgxcron
RUN mkdir -p /var/lib
EXPOSE 8035

CMD ["pgxcron", "-databases", "/etc/pgxcron/databases.toml", "-crontab", "/etc/pgxcron/crontab.toml","-historyfile", "/var/lib/pgxcronhistory.db", "-webport","8035"]
COPY databases.toml /etc/pgxcron/databases.toml
COPY crontab.toml /etc/pgxcron/crontab.toml
