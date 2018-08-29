package config

import (
	"fmt"
	"log"

	"github.com/maurodelazeri/lion/postgres"
)

// Cfg stores a config
var Cfg Config

// Config holds the venues individual config
type Config struct {
	Venues  []VenueConfig
	Enabled bool
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

// GetVenueConfig returns your venue configurations by its indivdual name
func (c *Config) GetVenueConfig(name string) (VenueConfig, error) {
	for i := range c.Venues {
		if c.Venues[i].Name == name {
			return c.Venues[i], nil
		}
	}
	return VenueConfig{}, fmt.Errorf("Venue %s: Not found", name)
}

func (c *Config) loadDatabaseConfig() error {
	venues := []VenueConfig{}
	if err := postgres.DB.Select(&venues, "SELECT v.venue_id,v.name,v.enabled,v.api_key,v.api_secret,v.passphrase,p.individual_connection,p.product,p.venue_product,p.minimum_order_size,p.step_size,p.maker_fee,p.taker_fee FROM venues v, venues_products p WHERE v.venue_id=p.venue_id"); err != nil {
		log.Fatal(err)
	}
	for i, p := range venues {
		log.Printf("%d => %v , %v", i, p.VenueID, string(p.Name))
	}
	return nil
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig() error {
	err := c.loadDatabaseConfig()
	if err != nil {
		return fmt.Errorf("Fatal error checking config values. Error: %s", err)
	}
	return nil
}

// GetConfig returns a pointer to a confiuration object
func GetConfig() *Config {
	return &Cfg
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
