from golang:latest as builder

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
RUN apt-get update && apt-get install -y musl-dev musl-tools

WORKDIR /pgxcron
COPY go.mod .
COPY go.sum .
COPY history_schema.sql .
COPY history_queries.sql .
COPY sqlc.yaml .
RUN go mod download
RUN sqlc generate
COPY . .
# Compile sqlite & postgres parser with musl libc to target alpine
# musl being linked statically means the entire binary is portable
RUN CGO=1 CC=musl-gcc go build --ldflags '-linkmode=external -extldflags=-static'

from alpine:latest

COPY --from=builder /pgxcron/pgxcron /bin/pgxcron
RUN mkdir -p /var/lib
EXPOSE 8035

CMD ["pgxcron", "-databases", "/etc/pgxcron/databases.toml", "-crontab", "/etc/pgxcron/crontab.toml","-historyfile", "/var/lib/pgxcronhistory.db", "-webport","8035"]
COPY databases.toml /etc/pgxcron/databases.toml
COPY crontab.toml /etc/pgxcron/crontab.toml
