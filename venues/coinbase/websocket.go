package coinbase

import (
	//"encoding/json"

	"errors"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	"github.com/maurodelazeri/lion/common"
	"github.com/maurodelazeri/lion/mongo"
	pbAPI "github.com/maurodelazeri/lion/protobuf/api"
	"github.com/maurodelazeri/lion/streaming/kafka/producer"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
)

// Subscribe subsribe public and private endpoints
func (r *WebsocketCoinbase) Subscribe(products []string) error {
	subscribe := Message{
		Type: "subscribe",
		Channels: []MessageChannel{
			MessageChannel{
				Name:       "ticker",
				ProductIDs: products,
			},
			MessageChannel{
				Name:       "full",
				ProductIDs: products,
			},
			MessageChannel{
				Name:       "level2",
				ProductIDs: products,
			},
		},
	}

	json, err := common.JSONEncode(subscribe)
	if err != nil {
		return err
	}
	err = r.Conn.WriteMessage(websocket.TextMessage, json)
	if err != nil {
		return err
	}
	return nil
}

// Close closes the underlying network connection without
// sending or waiting for a close frame.
func (r *WebsocketCoinbase) Close() {
	r.mu.Lock()
	if r.Conn != nil {
		r.Conn.Close()
	}
	r.isConnected = false
	r.mu.Unlock()
}

// WebsocketClient ...
func (r *WebsocketCoinbase) WebsocketClient() {

	if r.RecIntvlMin == 0 {
		r.RecIntvlMin = 2 * time.Second
	}

	if r.RecIntvlMax == 0 {
		r.RecIntvlMax = 30 * time.Second
	}

	if r.RecIntvlFactor == 0 {
		r.RecIntvlFactor = 1.5
	}

	if r.HandshakeTimeout == 0 {
		r.HandshakeTimeout = 2 * time.Second
	}

	r.dialer = websocket.DefaultDialer
	r.dialer.HandshakeTimeout = r.HandshakeTimeout

	// Start reading from the socket
	r.startReading()

	go func() {
		r.connect()
	}()

	// wait on first attempt
	time.Sleep(r.HandshakeTimeout)
}

// IsConnected returns the WebSocket connection state
func (r *WebsocketCoinbase) IsConnected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.isConnected
}

// CloseAndRecconect will try to reconnect.
func (r *WebsocketCoinbase) closeAndRecconect() {
	r.Close()
	go func() {
		r.connect()
	}()
}

// GetHTTPResponse returns the http response from the handshake.
// Useful when WebSocket handshake fails,
// so that callers can handle redirects, authentication, etc.
func (r *WebsocketCoinbase) GetHTTPResponse() *http.Response {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.httpResp
}

// GetDialError returns the last dialer error.
// nil on successful connection.
func (r *WebsocketCoinbase) GetDialError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.dialErr
}

func (r *WebsocketCoinbase) connect() {

	r.setReadTimeOut(1)

	bb := &backoff.Backoff{
		Min:    r.RecIntvlMin,
		Max:    r.RecIntvlMax,
		Factor: r.RecIntvlFactor,
		Jitter: true,
	}

	rand.Seed(time.Now().UTC().UnixNano())

	r.OrderBookMAP = make(map[string]map[float64]float64)
	r.LiveOrderBook = make(map[string]pbAPI.Orderbook)
	r.pairsMapping = make(map[string]string)

	venueArrayPairs := []string{}

	for _, sym := range r.subscribedPairs {
		r.LiveOrderBook[sym] = pbAPI.Orderbook{}
		r.OrderBookMAP[sym+"bids"] = make(map[float64]float64)
		r.OrderBookMAP[sym+"asks"] = make(map[float64]float64)
		venueArrayPairs = append(venueArrayPairs, r.base.VenueConfig.Get(r.base.GetName()).Products[sym].VenueName)
		r.pairsMapping[r.base.VenueConfig.Get(r.base.GetName()).Products[sym].VenueName] = sym
	}

	for {
		nextItvl := bb.Duration()

		wsConn, httpResp, err := r.dialer.Dial(websocketURL, r.reqHeader)

		r.mu.Lock()
		r.Conn = wsConn
		r.dialErr = err
		r.isConnected = err == nil
		r.httpResp = httpResp
		r.mu.Unlock()

		if err == nil {
			if r.base.Verbose {
				logrus.Printf("Dial: connection was successfully established with %s\n", websocketURL)
			}

			err = r.Subscribe(venueArrayPairs)
			if err != nil {
				logrus.Printf("Websocket subscription error: %s\n", err)
			}
			break
		} else {
			if r.base.Verbose {
				logrus.Println(err)
				logrus.Println("Dial: will try again in", nextItvl, "seconds.")
			}
		}

		time.Sleep(nextItvl)
	}
}

// startReading is a helper method for getting a reader
// using NextReader and reading from that reader to a buffer.
// If the connection is closed an error is returned
// startReading initiates a websocket client
// https://github.com/json-iterator/go
// I'm NOT USING THE json/encode due the lack of performance library currently in use json-iterator
func (r *WebsocketCoinbase) startReading() {
	go func() {
		for {
			select {
			default:
				if r.IsConnected() {
					err := errors.New("websocket: not connected")
					msgType, resp, err := r.Conn.ReadMessage()
					if err != nil {
						logrus.Error(r.base.Name, " problem to read: ", err)
						r.closeAndRecconect()
						continue
					}
					switch msgType {
					case websocket.TextMessage:
						data := Message{}
						err = ffjson.Unmarshal(resp, &data)
						if err != nil {
							logrus.Error(err)
							continue
						}
						product := r.pairsMapping[data.ProductID]

						if data.Type == "l2update" {
							//start := time.Now()
							liveBookMemomory := &r.LiveOrderBook
							refLiveBook := (*liveBookMemomory)[product]
							var wg sync.WaitGroup

							for _, data := range data.Changes {
								switch data[0] {
								case "buy":
									if data[2] == "0" {
										price := r.base.Strfloat(data[1])
										if _, ok := r.OrderBookMAP[product+"bids"][price]; ok {
											delete(r.OrderBookMAP[product+"bids"], price)
										}
									} else {
										price := r.base.Strfloat(data[1])
										amount := r.base.Strfloat(data[2])
										// ignore updates > than the current r.MaxLevelsOrderBook levels we manage
										totalLevels := len(refLiveBook.GetBids())
										if totalLevels == r.MaxLevelsOrderBook {
											if price < refLiveBook.Bids[totalLevels-1].Price {
												continue
											}
										}
										r.OrderBookMAP[product+"bids"][price] = amount
									}
								case "sell":
									if data[2] == "0" {
										price := r.base.Strfloat(data[1])
										if _, ok := r.OrderBookMAP[product+"asks"][price]; ok {
											delete(r.OrderBookMAP[product+"asks"], price)
										}
									} else {
										// ignore updates > than the current r.MaxLevelsOrderBook levels we manage
										price := r.base.Strfloat(data[1])
										amount := r.base.Strfloat(data[2])
										totalLevels := len(refLiveBook.GetAsks())
										if totalLevels == r.MaxLevelsOrderBook {
											if price > refLiveBook.Asks[totalLevels-1].Price {
												continue
											}
										}
										r.OrderBookMAP[product+"asks"][price] = amount
									}
								default:
									continue
								}
							}

							wg.Add(1)
							go func() {
								refLiveBook.Bids = []*pbAPI.Item{}
								for price, amount := range r.OrderBookMAP[product+"bids"] {
									refLiveBook.Bids = append(refLiveBook.Bids, &pbAPI.Item{Price: price, Amount: amount})
								}
								sort.Slice(refLiveBook.Bids, func(i, j int) bool {
									return refLiveBook.Bids[i].Price > refLiveBook.Bids[j].Price
								})
								wg.Done()
							}()

							wg.Add(1)
							go func() {
								refLiveBook.Asks = []*pbAPI.Item{}
								for price, amount := range r.OrderBookMAP[product+"asks"] {
									refLiveBook.Asks = append(refLiveBook.Asks, &pbAPI.Item{Price: price, Amount: amount})
								}
								sort.Slice(refLiveBook.Asks, func(i, j int) bool {
									return refLiveBook.Asks[i].Price < refLiveBook.Asks[j].Price
								})
								wg.Done()
							}()

							wg.Wait()

							wg.Add(1)
							go func() {
								totalBids := len(refLiveBook.Bids)
								if totalBids > r.MaxLevelsOrderBook {
									refLiveBook.Bids = refLiveBook.Bids[0:r.MaxLevelsOrderBook]
								}
								wg.Done()
							}()
							wg.Add(1)
							go func() {
								totalAsks := len(refLiveBook.Asks)
								if totalAsks > r.MaxLevelsOrderBook {
									refLiveBook.Asks = refLiveBook.Asks[0:r.MaxLevelsOrderBook]
								}
								wg.Done()
							}()

							wg.Wait()
							(*liveBookMemomory)[product] = refLiveBook

							book := &pbAPI.Orderbook{
								Product:   pbAPI.Product((pbAPI.Product_value[product])),
								Venue:     pbAPI.Venue((pbAPI.Venue_value[r.base.GetName()])),
								Levels:    int32(r.MaxLevelsOrderBook),
								Timestamp: r.base.MakeTimestamp(),
								Asks:      (*liveBookMemomory)[product].Asks,
								Bids:      (*liveBookMemomory)[product].Bids,
								VenueType: pbAPI.VenueType_SPOT,
							}
							serialized, err := proto.Marshal(book)
							if err != nil {
								log.Fatal("proto.Marshal error: ", err)
							}
							// for testing
							// if book.Product == pbAPI.Product_BTC_USD {
							// 	elapsed := time.Since(start)
							// 	// logrus.Info("Done nats ", elapsed)
							// 	logrus.Info("BEST BID: ", book.Bids[0].Price, " Size: ", book.Bids[0].Amount, " Best Ask: ", book.Asks[0].Price, " Size: ", book.Asks[0].Amount, " elapsed: ", elapsed)
							// }
							r.MessageType[0] = 1
							serialized = append(r.MessageType, serialized[:]...)
							kafkaproducer.PublishMessageAsync(product+"."+r.base.Name+".orderbook", serialized, 1, false)
							//mongodb.MongoQueue.Enqueue(book)
							//	elapsed := time.Since(start)
							//	logrus.Info("Done nats ", elapsed)
						}

						if data.Type == "match" {
							var side pbAPI.Side

							if data.Side == "buy" {
								side = pbAPI.Side_BUY
							} else {
								side = pbAPI.Side_SELL
							}

							trades := &pbAPI.Trade{
								Product:   pbAPI.Product((pbAPI.Product_value[product])),
								Venue:     pbAPI.Venue((pbAPI.Venue_value[r.base.GetName()])),
								Timestamp: r.base.MakeTimestamp(),
								Price:     data.Price,
								OrderSide: side,
								Size:      data.Size,
								VenueType: pbAPI.VenueType_SPOT,
							}

							serialized, err := proto.Marshal(trades)
							if err != nil {
								log.Fatal("proto.Marshal error: ", err)
							}
							r.MessageType[0] = 0
							serialized = append(r.MessageType, serialized[:]...)
							kafkaproducer.PublishMessageAsync(product+"."+r.base.Name+".trade", serialized, 1, false)
							mongodb.MongoQueue.Enqueue(trades)
						}

						if data.Type == "snapshot" {

							liveBookMemomory := &r.LiveOrderBook
							refLiveBook := (*liveBookMemomory)[product]
							var wg sync.WaitGroup

							wg.Add(1)
							go func(arr [][]string) {
								total := 0
								for _, line := range arr {
									price := r.base.Strfloat(line[0])
									amount := r.base.Strfloat(line[1])
									if total > r.MaxLevelsOrderBook {
										continue
									}
									refLiveBook.Asks = append(refLiveBook.Asks, &pbAPI.Item{Price: price, Amount: amount})
									r.OrderBookMAP[product+"bids"][price] = amount
									total++
								}
								wg.Done()
							}(data.Bids)

							wg.Add(1)
							go func(arr [][]string) {
								total := 0
								for _, line := range arr {
									price := r.base.Strfloat(line[0])
									amount := r.base.Strfloat(line[1])
									if total > r.MaxLevelsOrderBook {
										continue
									}
									refLiveBook.Bids = append(refLiveBook.Bids, &pbAPI.Item{Price: price, Amount: amount})
									r.OrderBookMAP[product+"asks"][price] = amount
									total++
								}
								wg.Done()
							}(data.Asks)
							wg.Wait()

							wg.Add(1)
							go func() {
								sort.Slice(refLiveBook.Bids, func(i, j int) bool {
									return refLiveBook.Bids[i].Price > refLiveBook.Bids[j].Price
								})
								wg.Done()
							}()

							wg.Add(1)
							go func() {
								sort.Slice(refLiveBook.Asks, func(i, j int) bool {
									return refLiveBook.Asks[i].Price < refLiveBook.Asks[j].Price
								})
								wg.Done()
							}()
							wg.Wait()

							(*liveBookMemomory)[product] = refLiveBook
						}
					}
				}
			}
		}
	}()
}

func (r *WebsocketCoinbase) setReadTimeOut(timeout int) {
	if r.Conn != nil && timeout != 0 {
		readTimeout := time.Duration(timeout) * time.Second
		r.Conn.SetReadDeadline(time.Now().Add(readTimeout))
	}

}
