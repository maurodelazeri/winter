package coinbase

import (
	"net/http"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gorilla/websocket"
	pbmarket "github.com/maurodelazeri/lion/protobuf/marketdata"
	"github.com/maurodelazeri/winter/config"
	venue "github.com/maurodelazeri/winter/venues"
)

const websocketURL = "wss://ws-feed.pro.coinbase.com"

// Coinbase internals
type Coinbase struct {
	venue.Base
	kafkaProducer *kafka.Producer
}

// WebsocketCoinbase is the overarching type across the Coinbase package
type WebsocketCoinbase struct {
	base        *Coinbase
	nonce       int64
	isConnected bool
	mu          sync.Mutex
	*websocket.Conn
	dialer    *websocket.Dialer
	reqHeader http.Header
	httpResp  *http.Response
	dialErr   error

	// RecIntvlMin specifies the initial reconnecting interval,
	// default to 2 seconds
	RecIntvlMin time.Duration
	// RecIntvlMax specifies the maximum reconnecting interval,
	// default to 30 seconds
	RecIntvlMax time.Duration
	// RecIntvlFactor specifies the rate of increase of the reconnection
	// interval, default to 1.5
	RecIntvlFactor float64
	// HandshakeTimeout specifies the duration for the handshake to complete,
	// default to 2 seconds
	HandshakeTimeout time.Duration

	OrderBookMAP    map[string]map[float64]float64
	LiveOrderBook   map[string]pbmarket.Orderbook
	subscribedPairs []string
	pairsMapping    map[string]string

	MessageType []byte
}

// SetDefaults sets default values for the venue
func (r *Coinbase) SetDefaults() {
	r.Enabled = true
}

// Setup initialises the venue parameters with the current configuration
func (r *Coinbase) Setup(venue string, products map[string]config.VenueConfig) {
	r.Name = venue
	r.Pairs = products
}

// Start ...
func (r *Coinbase) Start() {

	var dedicatedSocket, sharedSocket []string

	for key, value := range r.Pairs {
		if value.IndividualConnection {
			dedicatedSocket = append(dedicatedSocket, key)
		} else {
			sharedSocket = append(sharedSocket, key)
		}
	}

	if len(dedicatedSocket) > 0 {
		for _, pair := range dedicatedSocket {
			socket := new(WebsocketCoinbase)
			socket.MessageType = make([]byte, 4)
			socket.base = r
			socket.subscribedPairs = append(socket.subscribedPairs, pair)
			go socket.WebsocketClient()
		}
	}

	if len(sharedSocket) > 0 {
		socket := new(WebsocketCoinbase)
		socket.MessageType = make([]byte, 4)
		socket.base = r
		socket.subscribedPairs = sharedSocket
		go socket.WebsocketClient()
	}
}
