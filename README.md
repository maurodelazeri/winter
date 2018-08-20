# winter

Streaming data from venues to nats pipeline

```
export DB_HOST=127.0.0.1:3306
export MYSQLUSER=root
export MYSQLPASS=123456

export NATS_SERVER="nats://192.168.1.37:4222"
export BROKERS="192.168.1.37:9092"


export BROKERS="192.168.3.100:9092"
export DB_HOST=192.168.3.100:3306


CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

https://nats.io/documentation/server/gnatsd-authorization/
gnatsd -c server.cfg
gnatsd -m 8222
nats-top
```
