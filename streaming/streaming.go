package streaming

import (
	"os"

	"github.com/sirupsen/logrus"
	mangos "nanomsg.org/go-mangos"
	"nanomsg.org/go-mangos/protocol/pub"
	"nanomsg.org/go-mangos/transport/tcp"
)

// NanomsgServer stores the sessiion
type NanomsgServer struct {
	nanoMSG mangos.Socket
	Message chan []byte
}

// 99, 111, 105, 110, 98, 97, 115, 101, 58, 66, 84, 67, 85, 83, 68, 58, 111, 114, 100, 101, 114, 98, 111, 111, 107
// coinbase:BTCUSD:book

// Initialize nanomsg instance
func (r *NanomsgServer) Initialize(venue string) {

	r.Message = make(chan []byte, 10000)

	var url string
	switch venue {
	case "coinbase":
		url = os.Getenv("PUSH_COINBASE_BIND")
	case "binance":
		url = os.Getenv("PUSH_BINANCE_BIND")
	case "bitfinex":
		url = os.Getenv("PUSH_BITFINEX_BIND")
	case "bitmex":
		url = os.Getenv("PUSH_BITMEX_BIND")
	default:
		logrus.Warn("Nanomsg cant initialized, venue does not exist: ", venue)
		os.Exit(1)
	}
	socket, err := pub.NewSocket()
	if err != nil {
		logrus.Error("Problem to initialize NanoMSG: ", err)
		os.Exit(1)
	}
	//	p.NanoMSG.AddTransport(ipc.NewTransport())
	socket.AddTransport(tcp.NewTransport())
	err = socket.Listen(url)
	r.nanoMSG = socket

	if err != nil {
		logrus.Error("Problem bind NanoMSG: ", err)
		os.Exit(1)
	}

	r.SendMessage()
}

// SendMessage sends create a go routine to send messages in a buffered channel
func (r *NanomsgServer) SendMessage() {
	go func() {
		for {
			select {
			case p, ok := <-r.Message:
				if ok {
					if err := r.nanoMSG.Send(p); err != nil {
						logrus.Error("Failed publishing: ", err.Error())
					}
				} else {
					logrus.Error("Problem to read from channel")
				}
			default:
				//logrus.Error("Channel full. Discarding value")
			}
		}
	}()
}
