package config

import (
	"database/sql"
	"fmt"
	"strings"

	pbmarket "github.com/maurodelazeri/winter/marketdata"

	"github.com/maurodelazeri/lion/common"
	"github.com/maurodelazeri/lion/mysql"
	"github.com/sirupsen/logrus"
)

// Cfg stores a config
var Cfg Config

// Config holds the venues individual config
type Config struct {
	Venues       []VenueConfig
	mysqlSession *sql.DB
}

// VenueConfig holds all the information needed for each enabled Venue.
type VenueConfig struct {
	Name               string
	Enabled            bool
	Verbose            bool
	WebsocketDedicated []string
	VenueEnabledPairs  []string
	KafkaPartition     int32
	APIEnabledPairs    []string
	Products           map[string]pbmarket.Candle
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

// CheckVenueConfigValues returns configuation values for all enabled
// venues
func (c *Config) CheckVenueConfigValues() error {
	c.mysqlSession = mysqlserver.GetMysqlSession()
	sql := "SELECT e.name,e.enabled, kafka_partition, COALESCE((SELECT GROUP_CONCAT(pair) FROM venue_pair p WHERE e.id=p.venue_id and p.enabled=1),'0') as enabledPairs, COALESCE((SELECT GROUP_CONCAT(pair_venue) FROM venue_pair p WHERE e.id=p.venue_id and p.enabled=1),'0') as venuePairs, COALESCE((SELECT GROUP_CONCAT(pair_venue) FROM venue_pair p WHERE e.id=p.venue_id and p.enabled=1 and p.websocket_dedicated=1),'0') as websocketDedicated FROM venue e, venue_pair p group by e.id"
	rows, err := c.mysqlSession.Query(sql)

	checkErr(err)

	for rows.Next() {
		var name string
		var enabled bool
		var kafkaPartition int32
		var enabledPairs string
		var venuePairs string
		var websocketDedicated string

		err = rows.Scan(&name, &enabled, &kafkaPartition, &enabledPairs, &venuePairs, &websocketDedicated)
		checkErr(err)

		if enabledPairs == "" || venuePairs == "" {
			logrus.Error("Venue ", name, " pairs are not properly configured")
			continue
		}

		var venue VenueConfig

		if enabledPairs != "0" || venuePairs != "0" {
			venue.APIEnabledPairs = common.SplitStrings(strings.Replace(enabledPairs, " ", "", -1), ",")
			venue.VenueEnabledPairs = common.SplitStrings(strings.Replace(venuePairs, " ", "", -1), ",")
		}

		if websocketDedicated == "0" {
			venue.WebsocketDedicated = []string{}
		} else {
			venue.WebsocketDedicated = common.SplitStrings(strings.Replace(websocketDedicated, " ", "", -1), ",")
		}

		venue.Name = name
		venue.Enabled = enabled
		venue.KafkaPartition = kafkaPartition

		if enabled {
			logrus.Info("EXCHANGE: ", name, " ENABLED ", len(venue.APIEnabledPairs), " pairs")
		}

		c.Venues = append(c.Venues, venue)
	}

	if len(c.Venues) == 0 {
		logrus.Info("There is no venues enabled")
	}

	return nil
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig() error {
	err := c.CheckVenueConfigValues()
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
