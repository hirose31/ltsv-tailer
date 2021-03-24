package metrics

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gopkg.in/yaml.v1"
)

// Kind is a type of metrics.
type Kind int

const (
	// COUNTER is metrics.Counter.
	COUNTER Kind = iota
	// HISTOGRAM is metrics.Histogram
	HISTOGRAM
)

func (k Kind) String() string {
	switch k {
	case COUNTER:
		return "Counter"
	case HISTOGRAM:
		return "Histogram"
	default:
		return "Unknown"
	}
}

// Store is contains record and value transformer and metrics (Counter, Histogram).
type Store struct {
	RecordTransformer map[string]func(string) string
	ValueTransformer  map[string]func(float64) float64
	Counter           []*Counter
	Histogram         []*Histogram
}

// NewStore creates a new Store.
func NewStore() *Store {
	store := &Store{}
	store.RecordTransformer = map[string]func(string) string{}
	store.ValueTransformer = map[string]func(float64) float64{}
	return store
}

// Load loads from metrics config file.
func (store *Store) Load(conf string) {
	content, err := ioutil.ReadFile(filepath.Clean(conf))
	if err != nil {
		glog.Fatal(err)
	}

	var config Config
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		glog.Fatal(err)
	}

	for k, v := range config.Transform {
		switch k {
		case "tolower":
			for _, e := range v.([]interface{}) {
				store.RecordTransformer[e.(string)] = func(s string) string {
					return strings.ToLower(s)
				}
			}
		case "tosec":
			for _, e := range v.([]interface{}) {
				for ek, ev := range e.(map[interface{}]interface{}) {
					switch ev.(string) {
					case "microsec":
						store.ValueTransformer[ek.(string)] = func(f float64) float64 {
							return f / 1000000
						}
					case "millisec":
						store.ValueTransformer[ek.(string)] = func(f float64) float64 {
							return f / 1000
						}
					default:
						glog.Fatalf("unknown from tosec: %s", ev.(string))
					}
				}
			}
		}
	}

	for _, mr := range config.Metrics {
		labels := make([]string, len(mr["labels"].([]interface{})))
		for i, v := range mr["labels"].([]interface{}) {
			labels[i] = v.(string)
		}

		switch mr["kind"] {
		case "counter":
			store.Add(COUNTER,
				mr["value_key"].(string),
				mr["name"].(string),
				mr["help"].(string),
				nil,
				labels,
			)
		case "histogram":
			buckets := make([]float64, len(mr["buckets"].([]interface{})))
			for i, v := range mr["buckets"].([]interface{}) {
				buckets[i] = v.(float64)
			}
			store.Add(HISTOGRAM,
				mr["value_key"].(string),
				mr["name"].(string),
				mr["help"].(string),
				buckets,
				labels,
			)
		default:
			glog.Fatal("unknown metric kind: ", mr["kind"])
		}
	}
	glog.V(3).Infof("Store: %#v", store)
}

// Add specified metric to store
func (store *Store) Add(
	Kind Kind,
	ValueKey string,
	Name string,
	Help string,
	Buckets []float64,
	Labels []string,
) {
	glog.V(3).Infof("add %s %s", Kind, Name)
	switch Kind {
	case COUNTER:
		store.Counter = append(store.Counter, &Counter{
			Name:     Name,
			ValueKey: ValueKey,
			Labels:   Labels,
			Metric: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: Name,
					Help: Help,
				},
				Labels,
			),
		})
	case HISTOGRAM:
		store.Histogram = append(store.Histogram, &Histogram{
			Name:     Name,
			ValueKey: ValueKey,
			Labels:   Labels,
			Metric: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    Name,
					Help:    Help,
					Buckets: Buckets,
				},
				Labels,
			),
		})
	default:
		glog.Fatal("unknown kind: ", Kind)
	}
}

// Process record.
func (store *Store) Process(record map[string]string) {
METRIC_COUNTER:
	for _, metric := range store.Counter {
		glog.V(3).Infof("metric: %#v\n", metric)
		var val float64
		if metric.ValueKey == "COUNTER" {
			val = 1
		} else {
			var err error
			val, err = strconv.ParseFloat(record[metric.ValueKey], 64)
			if err != nil {
				glog.Warningf("SKIP failed to parse float: %#v", record[metric.ValueKey])
				continue METRIC_COUNTER
			}
		}

		labelKV := prometheus.Labels{}
		for _, key := range metric.Labels {
			if val, ok := record[key]; ok {
				labelKV[key] = val
			} else {
				glog.Warningf("SKIP missing key: %s", key)
				continue METRIC_COUNTER
			}
		}
		glog.V(3).Infof("%#v\n", labelKV)

		glog.V(3).Infof("Add %.6f to %s", val, metric.Name)
		metric.Metric.With(labelKV).Add(val)
	}

METRIC_HISTOGRAM:
	for _, metric := range store.Histogram {
		glog.V(3).Infof("metric: %#v\n", metric)
		var val float64
		var err error
		val, err = strconv.ParseFloat(record[metric.ValueKey], 64)
		glog.V(3).Infof("%#v\n", val)

		if err != nil {
			glog.Warningf("SKIP failed to parse float: %#v", record[metric.ValueKey])
			continue METRIC_HISTOGRAM
		}
		if transformer, ok := store.ValueTransformer[metric.ValueKey]; ok {
			val = transformer(val)
		}

		labelKV := prometheus.Labels{}
		for _, key := range metric.Labels {
			if val, ok := record[key]; ok {
				labelKV[key] = val
			} else {
				glog.Warningf("SKIP missing key: %s", key)
				continue METRIC_HISTOGRAM
			}
		}

		glog.V(3).Infof("Observe %.6f to %s", val, metric.Name)
		metric.Metric.With(labelKV).Observe(val)
	}
}
