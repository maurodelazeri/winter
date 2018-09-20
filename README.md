# winter

Streaming data from venues to nats pipeline

```
export KAFKA_BROKERS="192.168.3.100:9092"
export INFLUX_UDP_ADDR="192.168.3.100:8089"
export INFLUX_DATABASE="zinnion_streaming"

export PSQL_USER="postgres"
export PSQL_PASS="Br@sa154"
export PSQL_DB="elliptor"
export PSQL_HOST="68.169.103.45"
export PSQL_PORT="5432"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"'
```
