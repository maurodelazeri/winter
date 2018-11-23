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

```
export INFLUX_UDP_ADDR="192.168.1.200:8089"
export KAFKA_BROKERS="68.169.103.37:9092"
export MONGODB_DATABASE_NAME="zinnion"
export MONGODB_CONNECTION="winter.zinnion.com:27017"
export MONGODB_USERNAME="mongo-admin"
export MONGODB_PASSWORD="Br@sa154"
```

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"'
