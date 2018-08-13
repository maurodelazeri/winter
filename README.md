# winter

Streaming data from venues to nats pipeline

```
export DB_HOST=127.0.0.1:3306
export MYSQLUSER=root
export MYSQLPASS=123456
export PUSH_COINBASE_BIND="tcp://:2020"
export PUSH_BITFINEX_BIND="tcp://:2021"
export PUSH_BINANCE_BIND="tcp://:2022"
export PUSH_BITMEX_BIND="tcp://:2023"

export NATS_SERVER="nats://localhost:4222"

export BROKERS="127.0.0.1:9092"
export ACCOUNT_CURRENT_KEY="mauro"

export SERAPH_PUBSUB_URL="x"
export SERAPH_REQREP_URL="x"

export BROKERS="192.168.3.100:9092"
export DB_HOST=192.168.3.100:3306


CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

https://nats.io/documentation/server/gnatsd-authorization/
gnatsd -c server.cfg
gnatsd -m 8222
nats-top
```
