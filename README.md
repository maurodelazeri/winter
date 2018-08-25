# winter

Streaming data from venues to nats pipeline

```
export DB_HOST=192.168.1.36:3306
export MYSQLUSER=zinnion
export MYSQLPASS=Br@sa154
export STREAMING_SERVER="nats://192.168.1.37:4222"
export KAFKA_BROKERS="192.168.1.37:9092"

export DB_HOST=192.168.3.100:3306
export MYSQLUSER=root
export MYSQLPASS=123456
export STREAMING_SERVER="nats://192.168.3.100:4222"
export KAFKA_BROKERS="192.168.3.100:9092"


https://nats.io/documentation/server/gnatsd-authorization/
gnatsd -c server.cfg
gnatsd -m 8222
nats-top

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"' 

```
