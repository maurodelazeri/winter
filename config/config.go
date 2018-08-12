package config

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/maurodelazeri/winter/common"
	"github.com/maurodelazeri/winter/mysql"
	"github.com/sirupsen/logrus"
)

// Cfg stores a config
var Cfg Config

// Config holds the exchanges individual config
type Config struct {
	Exchanges    []ExchangeConfig
	mysqlSession *sql.DB
}

// ExchangeConfig holds all the information needed for each enabled Exchange.
type ExchangeConfig struct {
	Name                 string
	Enabled              bool
	Verbose              bool
	WebsocketDedicated   []string
	ExchangeEnabledPairs []string
	KafkaPartition       int32
	APIEnabledPairs      []string
}

// GetExchangeConfig returns your exchange configurations by its indivdual name
func (c *Config) GetExchangeConfig(name string) (ExchangeConfig, error) {
	for i := range c.Exchanges {
		if c.Exchanges[i].Name == name {
			return c.Exchanges[i], nil
		}
	}
	return ExchangeConfig{}, fmt.Errorf("Exchange %s: Not found", name)
}

// CheckExchangeConfigValues returns configuation values for all enabled
// exchanges
func (c *Config) CheckExchangeConfigValues() error {
	c.mysqlSession = mysqlserver.GetMysqlSession()
	sql := "SELECT e.name,e.enabled, kafka_partition, COALESCE((SELECT GROUP_CONCAT(pair) FROM exchange_pair p WHERE e.id=p.exchange_id and p.enabled=1),'0') as enabledPairs, COALESCE((SELECT GROUP_CONCAT(pair_exchange) FROM exchange_pair p WHERE e.id=p.exchange_id and p.enabled=1),'0') as exchangePairs, COALESCE((SELECT GROUP_CONCAT(pair_exchange) FROM exchange_pair p WHERE e.id=p.exchange_id and p.enabled=1 and p.websocket_dedicated=1),'0') as websocketDedicated FROM exchange e, exchange_pair p group by e.id"
	rows, err := c.mysqlSession.Query(sql)

	checkErr(err)

	for rows.Next() {
		var name string
		var enabled bool
		var kafkaPartition int32
		var enabledPairs string
		var exchangePairs string
		var websocketDedicated string

		err = rows.Scan(&name, &enabled, &kafkaPartition, &enabledPairs, &exchangePairs, &websocketDedicated)
		checkErr(err)

		if enabledPairs == "" || exchangePairs == "" {
			logrus.Error("Exchange ", name, " pairs are not properly configured")
			continue
		}

		var exchange ExchangeConfig

		if enabledPairs != "0" || exchangePairs != "0" {
			exchange.APIEnabledPairs = common.SplitStrings(strings.Replace(enabledPairs, " ", "", -1), ",")
			exchange.ExchangeEnabledPairs = common.SplitStrings(strings.Replace(exchangePairs, " ", "", -1), ",")
		}

		if websocketDedicated == "0" {
			exchange.WebsocketDedicated = []string{}
		} else {
			exchange.WebsocketDedicated = common.SplitStrings(strings.Replace(websocketDedicated, " ", "", -1), ",")
		}

		exchange.Name = name
		exchange.Enabled = enabled
		exchange.KafkaPartition = kafkaPartition

		if enabled {
			logrus.Info("EXCHANGE: ", name, " ENABLED ", len(exchange.APIEnabledPairs), " pairs")
		}

		c.Exchanges = append(c.Exchanges, exchange)
	}

	if len(c.Exchanges) == 0 {
		logrus.Info("There is no exchanges enabled")
	}

	return nil
}

// LoadConfig loads your configuration file into your configuration object
func (c *Config) LoadConfig() error {
	err := c.CheckExchangeConfigValues()
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
