# winter

Streaming data from venues to nats pipeline

```
export PSQL_HOST=68.169.103.38
export PSQL_USER="postgres"
export PSQL_PASS="Br@sa154"
export PSQL_DB="zinnion"
export SOCKET_SECRET_ADMIN="f21d3d09fd6f6e9631d855cc1fc0e36f"

export SOCKET_ADDR="68.169.103.39:8002"
export KAFKA_BROKERS="68.169.103.40:9092"
export SOCKET_SECRET="a69c81f636527a6ddbf99fecbf5b53d6"
export WINTER_VENUE_ID=16
export WINTER_DATAFEED_PRODUCTS=25
export WINTER_CONTAINER_EVENT="mauro local"

```

```
GOOS=linux go build -a -tags static_all -o build/winter

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"'
```

dockerd --tlsverify --tlscacert=ca.pem --tlscert=server-cert.pem --tlskey=server-key.pem -H=0.0.0.0:2376

GOOS=linux go build -a -tags static_all -o build/winter && scp build/winter mauro@zinnion.com:/www/zinnion-website/binaries/

bitcambio
export WINTER_VENUE_ID=16
export WINTER_DATAFEED_PRODUCTS=25
