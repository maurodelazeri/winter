package coinbase

import (
	//"encoding/json"

	"errors"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
	"github.com/maurodelazeri/winter/common"
	pb "github.com/maurodelazeri/winter/exchanges/proto"
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
	r.LiveOrderBook = make(map[string]pb.Orderbook)

	for _, sym := range r.subscribedPairs {
		position, exist := r.base.StringInSlice(sym, r.base.ExchangeEnabledPairs)
		symbol := strings.Split(r.base.APIEnabledPairs[position], "/")
		composedSymbol := symbol[0] + symbol[1]
		if exist {
			r.LiveOrderBook[composedSymbol] = pb.Orderbook{}
			r.OrderBookMAP[composedSymbol+"bids"] = make(map[float64]float64)
			r.OrderBookMAP[composedSymbol+"asks"] = make(map[float64]float64)
		}
	}

	for {
		nextItvl := bb.Duration()

		wsConn, httpResp, err := r.dialer.Dial(r.base.WebsocketURL, r.reqHeader)

		r.mu.Lock()
		r.Conn = wsConn
		r.dialErr = err
		r.isConnected = err == nil
		r.httpResp = httpResp
		r.mu.Unlock()

		if err == nil {
			if r.base.Verbose {
				logrus.Printf("Dial: connection was successfully established with %s\n", r.base.WebsocketURL)
			}

			err = r.Subscribe(r.subscribedPairs)
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
						position, exist := r.base.StringInSlice(data.ProductID, r.base.ExchangeEnabledPairs)
						if exist {
							symbolChannel := strings.Split(r.base.APIEnabledPairs[position], "/")
							composedSymbol := symbolChannel[0] + symbolChannel[1]

							if data.Type == "l2update" {
								//start := time.Now()

								liveBookMemomory := &r.LiveOrderBook
								refLiveBook := (*liveBookMemomory)[composedSymbol]
								var wg sync.WaitGroup

								for _, data := range data.Changes {
									switch data[0] {
									case "buy":
										if data[2] == "0" {
											price := r.base.Strfloat(data[1])
											if _, ok := r.OrderBookMAP[composedSymbol+"bids"][price]; ok {
												delete(r.OrderBookMAP[composedSymbol+"bids"], price)
											}
										} else {
											price := r.base.Strfloat(data[1])
											amount := r.base.Strfloat(data[2])
											r.OrderBookMAP[composedSymbol+"bids"][price] = amount
										}
									case "sell":
										if data[2] == "0" {
											price := r.base.Strfloat(data[1])
											if _, ok := r.OrderBookMAP[composedSymbol+"asks"][price]; ok {
												delete(r.OrderBookMAP[composedSymbol+"asks"], price)
											}
										} else {
											price := r.base.Strfloat(data[1])
											amount := r.base.Strfloat(data[2])
											r.OrderBookMAP[composedSymbol+"asks"][price] = amount
										}
									default:
										continue
									}
								}

								wg.Add(1)
								go func() {
									refLiveBook.Bids = []*pb.Item{}
									for price, amount := range r.OrderBookMAP[composedSymbol+"bids"] {
										refLiveBook.Bids = append(refLiveBook.Bids, &pb.Item{Price: price, Amount: amount})
									}
									sort.Slice(refLiveBook.Bids, func(i, j int) bool {
										return refLiveBook.Bids[i].Price > refLiveBook.Bids[j].Price
									})
									wg.Done()
								}()

								wg.Add(1)
								go func() {
									refLiveBook.Asks = []*pb.Item{}
									for price, amount := range r.OrderBookMAP[composedSymbol+"asks"] {
										refLiveBook.Asks = append(refLiveBook.Asks, &pb.Item{Price: price, Amount: amount})
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
									if totalBids > 20 {
										refLiveBook.Bids = refLiveBook.Bids[0:20]
									}
									wg.Done()
								}()
								wg.Add(1)
								go func() {
									totalAsks := len(refLiveBook.Asks)
									if totalAsks > 20 {
										refLiveBook.Asks = refLiveBook.Asks[0:20]
									}
									wg.Done()
								}()

								wg.Wait()
								(*liveBookMemomory)[composedSymbol] = refLiveBook

								book := &pb.Orderbook{
									Symbol:     composedSymbol,
									Levels:     20,
									Timestamp:  uint64(r.base.MakeTimestamp()),
									Asks:       (*liveBookMemomory)[composedSymbol].Asks,
									Bids:       (*liveBookMemomory)[composedSymbol].Bids,
									MarketType: pb.MarketDataType_SPOT,
								}
								serialized, err := proto.Marshal(book)
								if err != nil {
									log.Fatal("proto.Marshal error: ", err)
								}
								r.MessageType[0] = 2
								serialized = append(r.MessageType, serialized[:]...)
								r.base.natsProducer.PublishMessage(composedSymbol+"."+r.base.Name+".orderbook", serialized)
								//	elapsed := time.Since(start)
								//	logrus.Info("Done nats ", elapsed)
							}

							if data.Type == "match" {
								var side pb.Side
								if data.Side == "buy" {
									side = pb.Side_BUY
								} else {
									side = pb.Side_SELL
								}
								trades := &pb.Trade{
									Symbol:     composedSymbol,
									Timestamp:  uint64(r.base.MakeTimestamp()),
									Price:      data.Price,
									Side:       side,
									Size:       data.Price,
									MarketType: pb.MarketDataType_SPOT,
								}
								serialized, err := proto.Marshal(trades)
								if err != nil {
									log.Fatal("proto.Marshal error: ", err)
								}
								r.MessageType[0] = 1
								serialized = append(r.MessageType, serialized[:]...)
								r.base.natsProducer.PublishMessage(composedSymbol+"."+r.base.Name+".trade", serialized)
							}

							if data.Type == "ticker" {
								var side pb.Side
								if data.Side == "buy" {
									side = pb.Side_BUY
								} else {
									side = pb.Side_SELL
								}
								ticker := &pb.Ticker{
									Symbol:     composedSymbol,
									Timestamp:  uint64(r.base.MakeTimestamp()),
									Price:      data.Price,
									Side:       side,
									BestBid:    data.BestBid,
									BestAsk:    data.BestAsk,
									MarketType: pb.MarketDataType_SPOT,
								}
								serialized, err := proto.Marshal(ticker)
								if err != nil {
									log.Fatal("proto.Marshal error: ", err)
								}
								r.MessageType[0] = 0
								serialized = append(r.MessageType, serialized[:]...)
								r.base.natsProducer.PublishMessage(composedSymbol+"."+r.base.Name+".tick", serialized)
							}

							if data.Type == "snapshot" {

								liveBookMemomory := &r.LiveOrderBook
								refLiveBook := (*liveBookMemomory)[composedSymbol]
								var wg sync.WaitGroup

								wg.Add(1)
								go func(arr [][]string) {
									total := 0
									for _, line := range arr {
										price := r.base.Strfloat(line[0])
										amount := r.base.Strfloat(line[1])
										if total > 20 {
											continue
										}
										refLiveBook.Asks = append(refLiveBook.Asks, &pb.Item{Price: price, Amount: amount})
										r.OrderBookMAP[composedSymbol+"bids"][price] = amount
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
										if total > 20 {
											continue
										}
										refLiveBook.Bids = append(refLiveBook.Bids, &pb.Item{Price: price, Amount: amount})
										r.OrderBookMAP[composedSymbol+"asks"][price] = amount
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

								(*liveBookMemomory)[composedSymbol] = refLiveBook
							}
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
