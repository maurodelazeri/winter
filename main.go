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

	"github.com/maurodelazeri/winter/common"
	"github.com/maurodelazeri/winter/config"
	exchange "github.com/maurodelazeri/winter/exchanges"
	"github.com/maurodelazeri/winter/exchanges/coinbase"
	"github.com/sirupsen/logrus"
)

// Winter contains configuration
type Winter struct {
	exchanges []exchange.Winter
	config    *config.Config
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
	logrus.Infof("Loading config...")

	err := winter.config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	SetupExchanges()

	logrus.Infof("Winter started.\n")

	<-cancelContext.Done()
	fmt.Println("The cancel context has been cancelled...")

	Shutdown()
}

// SetupExchanges sets up the exchanges used by the westeros
func SetupExchanges() {
	for _, exch := range winter.config.Exchanges {
		if !exch.Enabled {
			log.Printf("%s: Exchange support: Disabled", exch.Name)
			continue
		} else {
			err := LoadExchange(exch.Name)
			if err != nil {
				log.Printf("LoadExchange %s failed: %s", exch.Name, err)
				continue
			}
		}
	}
}

// LoadExchange loads an exchange by name
func LoadExchange(name string) error {
	nameLower := common.StringToLower(name)
	var exch exchange.Winter
	switch nameLower {
	// case "bitfinex":
	// 	exch = new(bitfinex.Bitfinex)
	// case "bitmex":
	// 	exch = new(bitmex.Bitmex)
	// case "binance":
	// 	exch = new(binance.Binance)
	case "coinbase":
		exch = new(coinbase.Coinbase)
	default:
		return errors.New("exchange not found")
	}

	if exch == nil {
		return errors.New("exchange failed to load")
	}

	exch.SetDefaults()
	winter.exchanges = append(winter.exchanges, exch)
	exchCfg, err := winter.config.GetExchangeConfig(name)
	if err != nil {
		return err
	}
	exchCfg.Enabled = true
	exch.Setup(exchCfg)
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
