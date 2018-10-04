# winter

Streaming data from venues to nats pipeline

```
export KAFKA_BROKERS="68.169.103.37:9092"
export MONGODB_DATABASE_NAME="zinnion"
export MONGODB_CONNECTION_URL="mongodb://192.168.3.100:27017"

export MONGODB_CONNECTION_URL="mongodb://127.0.0.1:27017"


CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"'
```
