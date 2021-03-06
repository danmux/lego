package adapter

import (
	"fmt"
	"sort"
	"sync"

	"github.com/stairlin/lego/config"
	"github.com/stairlin/lego/log"
	"github.com/stairlin/lego/stats"
	"github.com/stairlin/lego/stats/adapter/influxdb"
)

func init() {
	Register(influxdb.Name, influxdb.New)
}

// Adapter returns a new store initialised with the given config
type Adapter func(config map[string]string) (stats.Stats, error)

// Void is a null stats adapter
type Void struct{}

func (s *Void) Start()                   {}
func (s *Void) Stop()                    {}
func (s *Void) Add(metric *stats.Metric) {}
func (s *Void) SetLogger(l log.Logger)   {}

func New(config *config.Stats) (stats.Stats, error) {
	if !config.On {
		return &Void{}, nil
	}

	return newStats(config.Adapter, config.Config)
}

var (
	adaptersMu sync.RWMutex
	adapters   = make(map[string]Adapter)
)

// Adapters returns the list of registered adapters
func Adapters() []string {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()

	var l []string
	for a := range adapters {
		l = append(l, a)
	}

	sort.Strings(l)

	return l
}

// Register makes a stats adapter available by the provided name.
// If an adapter is registered twice or if an adapter is nil, it will panic.
func Register(name string, adapter Adapter) {
	adaptersMu.Lock()
	defer adaptersMu.Unlock()

	if adapter == nil {
		panic("stats: Registered adapter is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("stats: Duplicated adapter")
	}

	adapters[name] = adapter
}

// NewStore returns a new stats instance
func newStats(adapter string, config map[string]string) (stats.Stats, error) {
	adaptersMu.RLock()
	defer adaptersMu.RUnlock()

	if f, ok := adapters[adapter]; ok {
		return f(config)
	}

	return nil, fmt.Errorf("stats adapter not found <%s>", adapter)
}
