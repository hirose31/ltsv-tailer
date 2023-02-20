// Package metrics provides Config struct
package metrics

// Config contains metrics and transform definitions.
type Config struct {
	Transform map[string]interface{}
	Metrics   []map[string]interface{}
}
