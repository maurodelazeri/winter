package coinbase

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/maurodelazeri/lion/streaming/nats/producer"
	"github.com/maurodelazeri/winter/config"
	venue "github.com/maurodelazeri/winter/venues"
	pb "github.com/maurodelazeri/winter/venues/proto"
)

// Coinbase internals
type Coinbase struct {
	venue.Base
	Name         string
	Enabled      bool
	Verbose      bool
	natsProducer *natsproducer.Producer
}

// WebsocketCoinbase is the overarching type across the Coinbase package
type WebsocketCoinbase struct {
	base            *Coinbase
	subscribedPairs []string

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

	OrderBookMAP  map[string]map[float64]float64
	LiveOrderBook map[string]pb.Orderbook

	MessageType []byte
}

// SetDefaults sets default values for the venue
func (r *Coinbase) SetDefaults() {
	r.Name = "coinbase"
}

// Setup initialises the venue parameters with the current configuration
func (r *Coinbase) Setup(exch config.VenueConfig) {
	if !exch.Enabled {
		r.SetEnabled(false)
	} else {
		r.Enabled = true
		r.Verbose = exch.Verbose
		r.APIEnabledPairs = exch.APIEnabledPairs
		r.VenueEnabledPairs = exch.VenueEnabledPairs
		r.WebsocketDedicated = exch.WebsocketDedicated
		r.KafkaPartition = exch.KafkaPartition
		producer := new(natsproducer.Producer)
		producer.Initialize()
		r.natsProducer = producer
	}
}

// Start ...
func (r *Coinbase) Start() {
	r.WebsocketURL = "wss://ws-feed.pro.coinbase.com"

	var dedicatedSocket, sharedSocket []string
	for _, pair := range r.VenueEnabledPairs {
		_, exist := r.StringInSlice(pair, r.WebsocketDedicated)
		if exist {
			dedicatedSocket = append(dedicatedSocket, pair)
		} else {
			sharedSocket = append(sharedSocket, pair)
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
