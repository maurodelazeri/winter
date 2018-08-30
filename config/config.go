package config

import (
	"log"

	"github.com/maurodelazeri/lion/postgres"
)

// Cfg stores a config
var Cfg Config

// Config holds the venues individual config
type Config struct {
	Venues map[string]map[string]VenueConfig
}

// VenueConfig holds all the information needed for each enabled Venue.
type VenueConfig struct {
	VenueID              int     `db:"venue_id"`
	Name                 string  `db:"name"`
	Enabled              bool    `db:"enabled"`
	APIKey               string  `db:"api_key"`
	APISecret            string  `db:"api_secret"`
	Passphrase           string  `db:"passphrase"`
	IndividualConnection bool    `db:"individual_connection"`
	Product              string  `db:"product"`
	VenueProduct         string  `db:"venue_product"`
	MinimumOrderSize     float64 `db:"minimum_order_size"`
	StepSize             float64 `db:"step_size"`
	MakerFee             float64 `db:"maker_fee"`
	TakerFee             float64 `db:"taker_fee"`
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig() error {
	venues := []VenueConfig{}
	if err := postgres.DB.Select(&venues, "SELECT v.venue_id,v.name,v.enabled,v.api_key,v.api_secret,v.passphrase,p.individual_connection,p.product,p.venue_product,p.minimum_order_size,p.step_size,p.maker_fee,p.taker_fee FROM venues v, venues_products p WHERE v.venue_id=p.venue_id"); err != nil {
		log.Fatal(err)
	}
	for _, p := range venues {
		c.Venues[p.Name] = make(map[string]VenueConfig)
	}
	for _, p := range venues {
		c.Venues[p.Name][p.Product] = p
	}
	return nil
}
