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

	"github.com/maurodelazeri/lion/mongo"
	venue "github.com/maurodelazeri/lion/venues"
	"github.com/maurodelazeri/lion/venues/coinbase"
	"github.com/maurodelazeri/lion/venues/config"
	"github.com/sirupsen/logrus"
)

// Winter contains configuration
type Winter struct {
	waitGroup sync.WaitGroup
	venues    *SyncMapVenuesConfig
	config    *config.Config
}

// SyncMapVenuesConfig ...
type SyncMapVenuesConfig struct {
	state map[string]venue.Venues
	mutex *sync.RWMutex
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

	logrus.Infof("Venues setup")
	winter.venues = NewSyncMapVenuesConfig()
	SetupVenues()

	winter.waitGroup.Wait()

	mongodb.InitQueueBacktesting()

	// logrus.Infof("Registering winter on kafka")
	// hostname, _ := os.Hostname()
	// appRegister, _ := ffjson.Marshal(&appInit{
	// 	Application: "winter",
	// 	Hostname:    hostname,
	// 	Timestamp:   common.MakeTimestamp(),
	// })
	// err = kafkaproducer.PublishMessageAsync("applications", appRegister, int32(0), true)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// logrus.Infof("Kafka is ready")

	logrus.Infof("Winter started \n")

	<-cancelContext.Done()
	fmt.Println("The cancel context has been cancelled...")

	Shutdown()
}

// SetupVenues sets up the venues used by the westeros
func SetupVenues() {
	for x := range winter.config.Venues.Values() {
		exch, err := LoadVenue(x)
		if err != nil {
			log.Printf("LoadVenue %s failed: %s", x.Name, err)
			continue
		}
		exch.SetDefaults()
		winter.venues.Put(x.Name, exch)
		exch.Setup(x.Name, x)
		exch.Start()
	}
}

// LoadVenue loads an venue by name
func LoadVenue(conf config.VenueConfig) (venue.Venues, error) {
	var exch venue.Venues
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
		return exch, errors.New("venue " + conf.Name + " not found")
	}
	if exch == nil {
		return exch, errors.New("venue failed to load")
	}
	return exch, nil
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

// NewSyncMapVenuesConfig ...
func NewSyncMapVenuesConfig() *SyncMapVenuesConfig {
	s := &SyncMapVenuesConfig{
		state: make(map[string]venue.Venues),
		mutex: &sync.RWMutex{},
	}
	return s
}

// Put ...
func (s *SyncMapVenuesConfig) Put(key string, value venue.Venues) {
	s.mutex.Lock()
	s.state[key] = value
	s.mutex.Unlock()
}

// Get ...
func (s *SyncMapVenuesConfig) Get(key string) venue.Venues {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.state[key]
}

// GetAllKeys ...
func (s *SyncMapVenuesConfig) GetAllKeys() map[string]venue.Venues {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.state
}

// Values ...
func (s *SyncMapVenuesConfig) Values() chan venue.Venues {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	values := make(chan venue.Venues, len(s.state))
	for _, value := range s.state {
		values <- value
	}
	close(values)
	return values
}
