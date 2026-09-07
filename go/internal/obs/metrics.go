// Package obs is a tiny dependency-free metrics registry exposing Prometheus text format.
package obs

import (
	"fmt"
	"sort"
	"sync"
)

// Metrics is a concurrency-safe counter/gauge registry.
type Metrics struct {
	mu       sync.Mutex
	counters map[string]float64
}

// New builds an empty registry.
func New() *Metrics { return &Metrics{counters: map[string]float64{}} }

// Inc adds delta to a counter.
func (m *Metrics) Inc(name string, delta float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += delta
}

// Get returns a counter's value.
func (m *Metrics) Get(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

// Expose renders the registry in Prometheus text format.
func (m *Metrics) Expose() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.counters))
	for n := range m.counters {
		names = append(names, n)
	}
	sort.Strings(names)
	var b []byte
	for _, n := range names {
		b = append(b, fmt.Sprintf("# TYPE %s counter\n%s %g\n", n, n, m.counters[n])...)
	}
	return string(b)
}
