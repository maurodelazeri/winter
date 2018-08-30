# winter

Streaming data from venues to nats pipeline

```
export STREAMING_SERVER="nats://127.0.0.1:4222"
export KAFKA_BROKERS="192.168.3.100:9092"

export PSQL_USER="postgres"
export PSQL_PASS="Br@sa154"
export PSQL_DB="elliptor"
export PSQL_HOST="68.169.103.45"
export PSQL_PORT="5432"


https://nats.io/documentation/server/gnatsd-authorization/
gnatsd -c server.cfg
gnatsd -m 8222
nats-top

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"'
```
