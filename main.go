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
	"strings"
	"sync"
	"syscall"
	"time"

	utils "github.com/maurodelazeri/concurrency-map-slice"
	event "github.com/maurodelazeri/lion/events"
	pbEvent "github.com/maurodelazeri/lion/protobuf/heraldsquareAPI"
	venue "github.com/maurodelazeri/lion/venues"
	"github.com/maurodelazeri/lion/venues/binance"
	"github.com/maurodelazeri/lion/venues/bitfinex"
	"github.com/maurodelazeri/lion/venues/coinbase"
	"github.com/maurodelazeri/lion/venues/config"
	"github.com/pquerna/ffjson/ffjson"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// Winter contains configuration
type Winter struct {
	waitGroup         sync.WaitGroup
	venues            *utils.ConcurrentMap
	config            *config.Config
	venuesInit        []string
	totalVenuesLoaded int
}

type appInit struct {
	Application string `json:"application,omitempty"`
	Hostname    string `json:"hostname,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
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
	winter.config.Venues = utils.NewConcurrentMap()

	err := winter.config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	logrus.Infof("Venues setup")
	winter.venues = utils.NewConcurrentMap()
	SetupVenues()

	winter.waitGroup.Wait()

	logrus.Infof("Registering summer on kafka")
	hostname, _ := os.Hostname()
	appRegister, _ := ffjson.Marshal(&appInit{
		Application: "winter",
		Hostname:    hostname,
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
	})
	eventID, _ := uuid.NewV4()
	eventData := event.CreateBaseEvent(eventID.String(), "main", appRegister, "", "", false, 0, pbEvent.System_ALASKA)
	event.PublishEvent(eventData, "events", int64(1), false)

	logrus.Info("Venues loaded ", winter.totalVenuesLoaded)

	if winter.totalVenuesLoaded == 0 {
		os.Exit(0)
	}

	logrus.Infof("Winter started \n")

	<-cancelContext.Done()
	fmt.Println("The cancel context has been cancelled...")

	Shutdown()
}

// SetupVenues sets up the venues used by the westeros
func SetupVenues() {
	for x := range winter.config.Venues.Iter() {
		data := x.Value
		venueConfig := data.(*config.VenueConfig)

		var found bool
		if len(winter.venuesInit) > 0 {
			for i := range winter.venuesInit {
				if strings.ToUpper(winter.venuesInit[i]) == venueConfig.Venue.Name {
					found = true
				}
			}
		}
		if !found {
			if len(winter.venuesInit) > 0 {
				continue
			}
		}
		exch, err := LoadVenue(*venueConfig)
		if err != nil {
			log.Printf("LoadVenue %s failed: %s", venueConfig.Venue.Name, err)
			continue
		}
		logrus.Info("Loading ", venueConfig.Venue.Name)
		exch.SetDefaults()
		winter.venues.Set(venueConfig.Venue.Name, exch)
		exch.Setup(venueConfig.Venue.Name, *venueConfig, true, 20)
		exch.Start()
		winter.totalVenuesLoaded++
	}
}

// LoadVenue loads an venue by name
func LoadVenue(conf config.VenueConfig) (venue.Venues, error) {
	var exch venue.Venues
	switch conf.Venue.Name {
	case "BINANCE":
		exch = new(binance.Binance)
	case "COINBASEPRO":
		exch = new(coinbase.Coinbase)
	case "BITFINEX":
		exch = new(bitfinex.Bitfinex)
	// case "BITMEX":
	// 	exch = new(bitmex.Bitmex)
	// case "OKEX_INTERNATIONAL_FUT":
	// 	exch = new(okex.Okex)
	// case "OKEX_INTERNATIONAL_SPOT":
	// 	exch = new(okex.Okex)
	// case "HUOBIPRO":
	// 	exch = new(huobi.Huobi)
	// case "DERIBIT":
	// 	exch = new(deribit.Deribit)
	// case "GATEIO":
	// 	exch = new(gateio.Gateio)
	// case "ZB":
	// 	exch = new(zb.Zb)
	// case "POLONIEX":
	// 	exch = new(poloniex.Poloniex)
	// case "GEMINI":
	// 	exch = new(gemini.Gemini)
	// case "FOXBIT":
	// 	exch = new(foxbit.Foxbit)
	// case "BITCAMBIO":
	// 	exch = new(bitcambio.Bitcambio)
	// case "BITCOINTOYOU":
	// 	exch = new(bitcointoyou.Bitcointoyou)
	default:
		return exch, errors.New("venue " + conf.Venue.Name + " not found")
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
