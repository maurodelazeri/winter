package venue

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/maurodelazeri/lion/orderbook"
	"github.com/maurodelazeri/winter/config"
)

// Base stores the individual venue information
type Base struct {
	Name            string
	Verbose         bool
	Enabled         bool
	SystemOrderbook map[string]*orderbook.OrderBook
	VenueConfig     *Internals
}

// Internals ...
type Internals struct {
	state map[string]config.VenueConfig
	mutex *sync.RWMutex
}

// Winter enforces standard functions for all venues supported in
type Winter interface {
	Setup(venue string, exch config.VenueConfig)
	SetDefaults()
	Start()
	GetName() string
	IsEnabled() bool
	SetEnabled(bool)
}

// NewInternals ...
func NewInternals() *Internals {
	s := &Internals{
		state: make(map[string]config.VenueConfig),
		mutex: &sync.RWMutex{},
	}
	return s
}

// Put ...
func (s *Internals) Put(key string, value config.VenueConfig) {
	s.mutex.Lock()
	s.state[key] = value
	s.mutex.Unlock()
}

// Get ...
func (s *Internals) Get(key string) config.VenueConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.state[key]
}

// Values ...
func (s *Internals) Values() chan config.VenueConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	values := make(chan config.VenueConfig, len(s.state))
	for _, value := range s.state {
		values <- value
	}
	close(values)
	return values
}

// GetName is a method that returns the name of the venue base
func (e *Base) GetName() string {
	return e.Name
}

// SetEnabled is a method that sets if the venue is enabled
func (e *Base) SetEnabled(enabled bool) {
	e.Enabled = enabled
}

// IsEnabled is a method that returns if the current venue is enabled
func (e *Base) IsEnabled() bool {
	return e.Enabled
}

// StringInSlice returns position of string, of exist
func (e *Base) StringInSlice(a string, list []string) (int, bool) {
	for index, b := range list {
		if b == a {
			return index, true
		}
	}
	return 0, false
}

// FloatInSlice returns position of float64, of exist
func (e *Base) FloatInSlice(a float64, list []float64) (int, bool) {
	for index, b := range list {
		if b == a {
			return index, true
		}
	}
	return 0, false
}

// IntnSlice returns position of string, of exist
func (e *Base) IntnSlice(a int64, list []int64) (int, bool) {
	for index, b := range list {
		if b == a {
			return index, true
		}
	}
	return 0, false
}

// RemoveIndex removes an index from a float array
func (e *Base) RemoveIndex(s []float64, index int) []float64 {
	return append(s[:index], s[index+1:]...)
}

// FloatToString to convert a float number to a string
func (e *Base) FloatToString(number float64) string {
	return strconv.FormatFloat(number, 'f', -1, 64)
}

// PercentChange gives the difference in percentage from 2 numbers
func (e *Base) PercentChange(base float64, current float64) float64 {
	if base > 0 && current > 0 {
		absolute := base - current
		if absolute < 0 {
			absolute = absolute * -1
		}
		average := ((base + current) / 2) / 2
		result := absolute / average
		formated := fmt.Sprintf("%.4f", result)
		return e.Strfloat(formated)
	}
	return -1
}

// Let's use 20 and 30 as an example. We will need to divide the absolute difference by the average of those two numbers and express it as percentages.

// Strfloat find the absolute difference between the two: |20 - 30| = |-10| = 10
// find the average of those two numbers: (20 + 30) / 2 = 50 / 2 = 25
// divide those two: 10 / 25 = 0.4
// express it as percentages: 0.4 * 100 = 40%
func (e *Base) Strfloat(i string) float64 {
	f, _ := strconv.ParseFloat(i, 64)
	return f
}
