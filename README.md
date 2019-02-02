# winter

Streaming data from venues to nats pipeline

```
export INFLUX_UDP_ADDR="192.168.1.200:8089"
export KAFKA_BROKERS="192.168.1.201:9092"
export MONGODB_DATABASE_NAME="zinnion"
export MONGODB_CONNECTION="192.168.1.100:27017"
export MONGODB_USERNAME="mongo-admin"
export MONGODB_PASSWORD="Br@sa154"
```

export PSQL_HOST=68.169.103.38
export PSQL_USER="postgres"
export PSQL_PASS="Br@sa154"
export PSQL_DB="zinnion"
export KAFKA_BROKERS="68.169.103.38:9092"
export SOCKET_URL="ws://68.169.103.39:8002/connection/websocket"
export=SOCKET_SECRET="Br@sa154"

```

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"'
```
