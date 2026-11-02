package httpapi

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const prometheusContentType = "text/plain; version=0.0.4; charset=utf-8"

func wantsPrometheus(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "prometheus")
}

func writeMetricFamily(b *strings.Builder, name, help, kind string, values map[string]int, label string) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, kind)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if label == "" {
			fmt.Fprintf(b, "%s %d\n", name, values[key])
			continue
		}
		fmt.Fprintf(b, "%s{%s=%q} %d\n", name, label, key, values[key])
	}
}

func writeScalarMetric(b *strings.Builder, name, help, kind string, value int) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s %s\n%s %d\n", name, help, name, kind, name, value)
}
