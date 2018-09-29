package config

import (
	"context"
	"log"
	"sync"

	"github.com/maurodelazeri/lion/mongo"
	"github.com/mongodb/mongo-go-driver/bson"
)

// Cfg stores a config
var Cfg Config

// Config holds the venues individual config
type Config struct {
	Venues *Internals
}

// Internals ...
type Internals struct {
	state map[string]VenueConfig
	mutex *sync.RWMutex
}

// DBconfig ...
type DBconfig struct {
	Protobuf   int32  `bson:"protobuf"`
	Enabled    bool   `bson:"enabled"`
	APIKey     string `bson:"api_key"`
	APISecret  string `bson:"api_secret"`
	Passphrase string `bson:"passphrase"`
	Name       string `bson:"name"`
	Product    []struct {
		Product              int32   `bson:"product"`
		Enabled              bool    `bson:"enabled"`
		IndividualConnection bool    `bson:"individual_connection"`
		VenueName            string  `bson:"venue_name"`
		APIName              string  `bson:"api_name"`
		MinimumOrdersSize    float64 `bson:"minimum_orders_size"`
		StepSize             float64 `bson:"step_size"`
		MakerFee             float64 `bson:"maker_fee"`
		TakerFee             float64 `bson:"taker_fee"`
	} `json:"product"`
}

// Product ...
type Product struct {
	Product              int32
	Enabled              bool
	IndividualConnection bool
	VenueName            string
	APIName              string
	MinimumOrdersSize    float64
	StepSize             float64
	MakerFee             float64
	TakerFee             float64
}

// VenueConfig holds all the information needed for each enabled Venue.
type VenueConfig struct {
	Protobuf   int32
	Enabled    bool
	APIKey     string
	APISecret  string
	Passphrase string
	Name       string
	Products   map[string]Product
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig() error {
	coll := mongodb.MongoDB.Collection("venue_products")
	filter := bson.NewDocument(bson.EC.Boolean("enabled", true))
	cursor, err := coll.Find(context.Background(), filter)
	if err != nil {
		return err
	}
	for cursor.Next(context.Background()) {
		var item DBconfig
		venueConf := make(map[string]Product)
		if err := cursor.Decode(&item); err != nil {
			log.Fatal(err)
		}
		for _, prods := range item.Product {
			if prods.Enabled {
				venueConf[prods.APIName] = Product{
					IndividualConnection: prods.IndividualConnection,
					VenueName:            prods.VenueName,
					APIName:              prods.APIName,
					MinimumOrdersSize:    prods.MinimumOrdersSize,
					StepSize:             prods.StepSize,
					MakerFee:             prods.MakerFee,
					TakerFee:             prods.TakerFee,
				}
			}
		}
		finalConf := VenueConfig{
			Protobuf:   item.Protobuf,
			Enabled:    item.Enabled,
			APIKey:     item.APIKey,
			APISecret:  item.APISecret,
			Passphrase: item.Passphrase,
			Name:       item.Name,
			Products:   venueConf,
		}
		Cfg.Venues.Put(item.Name, finalConf)
	}
	return nil
}

// NewInternals ...
func NewInternals() *Internals {
	s := &Internals{
		state: make(map[string]VenueConfig),
		mutex: &sync.RWMutex{},
	}
	return s
}

// Put ...
func (s *Internals) Put(key string, value VenueConfig) {
	s.mutex.Lock()
	s.state[key] = value
	s.mutex.Unlock()
}

// Get ...
func (s *Internals) Get(key string) VenueConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.state[key]
}

// Values ...
func (s *Internals) Values() chan VenueConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	values := make(chan VenueConfig, len(s.state))
	for _, value := range s.state {
		values <- value
	}
	close(values)
	return values
}
