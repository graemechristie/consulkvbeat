package kv

import (
	"regexp"

	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/hashicorp/consul/api"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("consulkv", "kv", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	keys []string
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct {
		Keys []string `config:"keys"`
	}{
		Keys: []string{},
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	logp.Warn("EXPERIMENTAL: The consulkv kv metricset is experimental %v", config)

	return &MetricSet{
		BaseMetricSet: base,
		keys:          config.Keys,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	config := api.DefaultConfig()
	config.Address = m.Host()

	client, err := api.NewClient(config)
	if err != nil {
		panic(err)
	}

	kv := client.KV()

	mapStrs := []common.MapStr{}
	for _, key := range m.keys {

		r, _ := regexp.Compile("[^(]*")
		base := r.FindString(key)

		result, _, err := kv.List(base, nil)

		if err != nil {
			return nil, err
		}

		keyRegex, _ := regexp.Compile(key)
		for _, pair := range result {
			mapStr := common.MapStr{}
			mapStr.Put("key", pair.Key)
			if keyRegex.MatchString(pair.Key) {
				match := keyRegex.FindStringSubmatch(pair.Key)
				for i, name := range keyRegex.SubexpNames() {
					if i != 0 {
						mapStr.Put(name, match[i])
					}
				}
				var dat map[string]interface{}
				if err := json.Unmarshal(pair.Value, &dat); err != nil {
					panic(err)
				}

				mapStr.Put("value", dat)
			}
			mapStrs = append(mapStrs, mapStr)
		}
	}

	return mapStrs, nil
}
