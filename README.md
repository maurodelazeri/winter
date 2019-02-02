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
export SOCKET_URL="ws://68.169.103.39:8002/connection/websocket?format=protobuf"
export SOCKET_SECRET="a69c81f636527a6ddbf99fecbf5b53d6"
export SOCKET_ADMIN_SECRET="f21d3d09fd6f6e9631d855cc1fc0e36f"

```

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"'
```
