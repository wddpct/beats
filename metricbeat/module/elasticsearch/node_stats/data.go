// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package node_stats

import (
	"encoding/json"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"jvm": c.Dict("jvm", s.Schema{
			"mem": c.Dict("mem", s.Schema{
				"pools": c.Dict("pools", s.Schema{
					"young":    c.Dict("young", poolSchema),
					"survivor": c.Dict("survivor", poolSchema),
					"old":      c.Dict("old", poolSchema),
				}),
			}),
			"gc": c.Dict("gc", s.Schema{
				"collectors": c.Dict("collectors", s.Schema{
					"young": c.Dict("young", collectorSchema),
					"old":   c.Dict("old", collectorSchema),
				}),
			}),
		}),
		"indices": c.Dict("indices", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"count":   c.Int("count"),
				"deleted": c.Int("deleted"),
			}),
			"store": c.Dict("store", s.Schema{
				"size": s.Object{
					"bytes": c.Int("size_in_bytes"),
				},
			}),
			"segments": c.Dict("segments", s.Schema{
				"count": c.Int("count"),
				"memory": s.Object{
					"bytes": c.Int("memory_in_bytes"),
				},
			}),
		}),
		"fs": c.Dict("fs", s.Schema{
			"summary": c.Dict("total", s.Schema{
				"total": s.Object{
					"bytes": c.Int("total_in_bytes"),
				},
				"free": s.Object{
					"bytes": c.Int("free_in_bytes"),
				},
				"available": s.Object{
					"bytes": c.Int("available_in_bytes"),
				},
			}),
		}),
	}

	poolSchema = s.Schema{
		"used": s.Object{
			"bytes": c.Int("used_in_bytes"),
		},
		"max": s.Object{
			"bytes": c.Int("max_in_bytes"),
		},
		"peak": s.Object{
			"bytes": c.Int("peak_used_in_bytes"),
		},
		"peak_max": s.Object{
			"bytes": c.Int("peak_max_in_bytes"),
		},
	}

	collectorSchema = s.Schema{
		"collection": s.Object{
			"count": c.Int("collection_count"),
			"ms":    c.Int("collection_time_in_millis"),
		},
	}
)

type nodesStruct struct {
	ClusterName string                            `json:"cluster_name"`
	Nodes       map[string]map[string]interface{} `json:"nodes"`
}

func eventsMapping(r mb.ReporterV2, content []byte) error {

	nodeData := &nodesStruct{}
	err := json.Unmarshal(content, nodeData)
	if err != nil {
		err = errors.Wrap(err, "failure parsing Elasticsearch Node Stats API response")
		r.Error(err)
		return err
	}

	var errs multierror.Errors
	for name, node := range nodeData.Nodes {
		event := mb.Event{}

		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", elasticsearch.ModuleName)

		event.ModuleFields = common.MapStr{
			"node": common.MapStr{
				"name": name,
			},
			"cluster": common.MapStr{
				"name": nodeData.ClusterName,
			},
		}

		event.MetricSetFields, err = schema.Apply(node)
		if err != nil {
			event.Error = errors.Wrap(err, "failure to apply node schema")
			r.Event(event)
			errs = append(errs, event.Error)
		}

		r.Event(event)
	}
	return errs.Err()
}
