//

package metrics

import (
	"encoding/json"
	"math"
	"sync"
)

// Gauge represents a metric of type Gauge. An example would be number of
// users logged in at the moment.
type Gauge struct {
	v  float64
	mu sync.RWMutex
}

// NewGauge initializes a Gauge and returns it
func NewGauge() *Gauge {
	g := new(Gauge)
	g.Reset()
	return g
}

// Reset - resets all values are reset to defaults
// Usually called from NewGauge but useful if you have to
// re-use and existing object
func (g *Gauge) Reset() {
	g.v = math.NaN()
}

// Set - Sets value of Gauge to input
func (g *Gauge) Set(v float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.v = v
}

// Get - Returns current value of Gauge
func (g *Gauge) Get() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.v
}

// MarshalJSON returns a byte slice of JSON representation of
// Gauge
func (g *Gauge) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.v)
}
