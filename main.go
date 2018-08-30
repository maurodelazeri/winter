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
	"syscall"

	"github.com/maurodelazeri/winter/config"
	venue "github.com/maurodelazeri/winter/venues"
	"github.com/maurodelazeri/winter/venues/coinbase"
	"github.com/sirupsen/logrus"
)

// Winter contains configuration
type Winter struct {
	venues map[string]*venue.Winter
	config *config.Config
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
	winter.config.Venues = make(map[string]map[string]config.VenueConfig)

	err := winter.config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	SetupVenues()

	logrus.Infof("Winter started.\n")

	<-cancelContext.Done()
	fmt.Println("The cancel context has been cancelled...")

	Shutdown()
}

// SetupVenues sets up the venues used by the westeros
func SetupVenues() {
	for x := range winter.config.Venues {
		err := LoadVenue(x)
		if err != nil {
			log.Printf("LoadVenue %s failed: %s", x, err)
			continue
		}

	}
}

// LoadVenue loads an venue by name
func LoadVenue(name string) error {
	var exch venue.Winter
	switch name {
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
		return errors.New("venue " + name + " not found")
	}

	if exch == nil {
		return errors.New("venue failed to load")
	}

	exch.SetDefaults()
	winter.venues[name] = &exch
	exch.Setup(name, winter.config.Venues[name])
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

// Shutdown correctly shuts down winter saving configuration files
func Shutdown() {
	logrus.Info("Winter shutting down..")
	logrus.Info("Exiting.")
	os.Exit(1)
}
