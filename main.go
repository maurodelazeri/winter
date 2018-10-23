package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"

	"github.com/maurodelazeri/lion/common"
	"github.com/maurodelazeri/lion/mongo"
	"github.com/maurodelazeri/lion/streaming/kafka/producer"
	"github.com/maurodelazeri/winter/config"
	venue "github.com/maurodelazeri/winter/venues"
	"github.com/maurodelazeri/winter/venues/coinbase"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
)

// Winter contains configuration
type Winter struct {
	waitGroup sync.WaitGroup
	venues    *WinterSyncMapConfig
	config    *config.Config
}

// WinterSyncMapConfig ...
type WinterSyncMapConfig struct {
	state map[string]venue.Winter
	mutex *sync.Mutex
}

type appInit struct {
	Application string `json:"application,omitempty"`
	Hostname    string `json:"hostname,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
}

const banner = `
██╗    ██╗██╗███╗   ██╗████████╗███████╗██████╗ 
██║    ██║██║████╗  ██║╚══██╔══╝██╔════╝██╔══██╗
██║ █╗ ██║██║██╔██╗ ██║   ██║   █████╗  ██████╔╝
██║███╗██║██║██║╚██╗██║   ██║   ██╔══╝  ██╔══██╗
╚███╔███╔╝██║██║ ╚████║   ██║   ███████╗██║  ██║
 ╚══╝╚══╝ ╚═╝╚═╝  ╚═══╝   ╚═╝   ╚══════╝╚═╝  ╚═╝                                                                                 
`

var winter Winter

func main() {

	HandleInterrupt()

	fmt.Println(banner)

	AdjustGoMaxProcs()

	cancelContext, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	winter.config = &config.Cfg
	logrus.Infof("Loading venues and products...")
	winter.config.Venues = config.NewInternals()

	err := winter.config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	logrus.Info("Init mongo queue")
	mongodb.InitQueueTrades()

	logrus.Infof("Venues setup")
	winter.venues = NewWinterSyncMapConfig()
	SetupVenues()

	// logrus.Infof("GRPC setup...")
	// go SetupGrpcServer()

	winter.waitGroup.Wait()

	logrus.Infof("Registering winter on kafka")
	hostname, _ := os.Hostname()
	appRegister, _ := ffjson.Marshal(&appInit{
		Application: "winter",
		Hostname:    hostname,
		Timestamp:   common.MakeTimestamp(),
	})
	err = kafkaproducer.PublishMessageAsync("applications", appRegister, int32(0), true)
	if err != nil {
		log.Fatal(err)
	}
	logrus.Infof("Kafka is ready")

	logrus.Infof("Winter started \n")

	<-cancelContext.Done()
	fmt.Println("The cancel context has been cancelled...")

	Shutdown()
}

// SetupVenues sets up the venues used by the westeros
func SetupVenues() {
	for x := range winter.config.Venues.Values() {
		err := LoadVenue(x)
		if err != nil {
			log.Printf("LoadVenue %s failed: %s", x.Name, err)
			continue
		}
	}
}

// SetupGrpcServer sets up the gRPC server
// func SetupGrpcServer() {
// 	lis, err := net.Listen("tcp", os.Getenv("GRPC_SERVER_PORT"))
// 	if err != nil {
// 		log.Fatalf("failed to listen: %v", err)
// 	}
// 	// Creates a new gRPC server
// 	s := grpc.NewServer()
// 	pb.RegisterAPIServer(s, &Winter{})

// 	winter.waitGroup.Done()
// 	s.Serve(lis)
// }

// LoadVenue loads an venue by name
func LoadVenue(conf config.VenueConfig) error {
	var exch venue.Winter
	switch conf.Name {
	// case "bitfinex":
	// 	exch = new(bitfinex.Bitfinex)
	// case "bitmex":
	// 	exch = new(bitmex.Bitmex)
	// case "binance":
	// 	exch = new(binance.Binance)
	// case "coinbase":
	// 	exch = new(coinbase.Coinbase)
	case "COINBASEPRO":
		exch = new(coinbase.Coinbase)
	default:
		return errors.New("venue " + conf.Name + " not found")
	}

	if exch == nil {
		return errors.New("venue failed to load")
	}

	exch.SetDefaults()
	winter.venues.Put(conf.Name, exch)
	exch.Setup(conf.Name, winter.config.Venues.Get(conf.Name))
	exch.Start()

	return nil
}

// AdjustGoMaxProcs adjusts the maximum processes that the CPU can handle.
func AdjustGoMaxProcs() {
	logrus.Info("Adjusting winter runtime performance..")
	maxProcsEnv := os.Getenv("GOMAXPROCS")
	maxProcs := runtime.NumCPU()
	logrus.Info("Number of CPU's detected:", maxProcs)

	if maxProcsEnv != "" {
		logrus.Info("GOMAXPROCS env =", maxProcsEnv)
		env, err := strconv.Atoi(maxProcsEnv)
		if err != nil {
			logrus.Info("Unable to convert GOMAXPROCS to int, using", maxProcs)
		} else {
			maxProcs = env
		}
	}
	if i := runtime.GOMAXPROCS(maxProcs); i != maxProcs {
		log.Fatal("Go Max Procs were not set correctly.")
	}
	logrus.Info("Set GOMAXPROCS to:", maxProcs)
}

// HandleInterrupt monitors and captures the SIGTERM in a new goroutine then
// shuts down winter
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		logrus.Infof("Captured %v.", sig)
		Shutdown()
	}()
}

// Shutdown correctly
func Shutdown() {
	logrus.Info("Winter shutting down..")
	logrus.Info("Exiting")
	os.Exit(1)
}

// NewWinterSyncMapConfig ...
func NewWinterSyncMapConfig() *WinterSyncMapConfig {
	s := &WinterSyncMapConfig{
		state: make(map[string]venue.Winter),
		mutex: &sync.Mutex{},
	}
	return s
}

// Put ...
func (s *WinterSyncMapConfig) Put(key string, value venue.Winter) {
	s.mutex.Lock()
	s.state[key] = value
	s.mutex.Unlock()
}

// Get ...
func (s *WinterSyncMapConfig) Get(key string) venue.Winter {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.state[key]
}

// Values ...
func (s *WinterSyncMapConfig) Values() chan venue.Winter {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	values := make(chan venue.Winter, len(s.state))
	for _, value := range s.state {
		values <- value
	}
	close(values)
	return values
}
