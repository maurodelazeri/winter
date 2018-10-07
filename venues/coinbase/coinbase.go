package coinbase

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/maurodelazeri/concurrency-map-slice"
	"github.com/maurodelazeri/lion/orderbook"
	"github.com/maurodelazeri/winter/config"
	venue "github.com/maurodelazeri/winter/venues"
)

const websocketURL = "wss://ws-feed.pro.coinbase.com"

// Coinbase internals
type Coinbase struct {
	venue.Base
	//	mutex sync.RWMutex
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

	OrderBookMAP       map[string]map[float64]float64
	LiveOrderBook      *utils.ConcurrentMap
	subscribedPairs    []string
	pairsMapping       *utils.ConcurrentMap
	MessageType        []byte
	MaxLevelsOrderBook int
}

// SetDefaults sets default values for the venue
func (r *Coinbase) SetDefaults() {
	r.Enabled = true
}

// Setup initialises the venue parameters with the current configuration
func (r *Coinbase) Setup(venueName string, config config.VenueConfig) {
	r.Name = venueName
	r.VenueConfig = utils.NewConcurrentMap()
	r.VenueConfig.Set(venueName, config)
}

// Start ...
func (r *Coinbase) Start() {
	var dedicatedSocket, sharedSocket []string
	// Individual system order book for each product
	r.SystemOrderbook = make(map[string]*orderbook.OrderBook)
	venueConf, ok := r.VenueConfig.Get(r.GetName())
	if ok {
		for product, value := range venueConf.(config.VenueConfig).Products {
			r.SystemOrderbook[product] = orderbook.NewOrderBook()
			// Separate products that will use a exclusive connection from those sharing a connection
			if value.IndividualConnection {
				dedicatedSocket = append(dedicatedSocket, product)
			} else {
				sharedSocket = append(sharedSocket, product)
			}
		}
		if len(dedicatedSocket) > 0 {
			for _, pair := range dedicatedSocket {
				socket := new(WebsocketCoinbase)
				socket.base = r
				socket.subscribedPairs = append(socket.subscribedPairs, pair)
				socket.MessageType = make([]byte, 4)
				socket.MaxLevelsOrderBook = 20
				go socket.WebsocketClient()
			}
		}
		if len(sharedSocket) > 0 {
			socket := new(WebsocketCoinbase)
			socket.base = r
			socket.subscribedPairs = sharedSocket
			socket.MessageType = make([]byte, 4)
			socket.MaxLevelsOrderBook = 20
			go socket.WebsocketClient()
		}
	}
}
