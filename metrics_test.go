package pid

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_Update(t *testing.T) {
	tests := []struct {
		name    string
		value   func(m *metrics) float64
		target  float64
		current float64
		want    float64
	}{
		{
			name:    "pid_updates_total",
			value:   func(m *metrics) float64 { return testutil.ToFloat64(m.updatesTotal) },
			target:  5.0,
			current: 2.0,
			want:    1.0,
		},
		{
			name:    "pid_target",
			value:   func(m *metrics) float64 { return testutil.ToFloat64(m.target) },
			target:  5.0,
			current: 2.0,
			want:    5.0,
		},
		{
			name:    "pid_current",
			value:   func(m *metrics) float64 { return testutil.ToFloat64(m.current) },
			target:  5.0,
			current: 2.0,
			want:    2.0,
		},
		{
			name:    "pid_control_signal",
			value:   func(m *metrics) float64 { return testutil.ToFloat64(m.controlSignal) },
			target:  5.0,
			current: 1.0,
			want:    4.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			controller, err := New(WithPrometheusMetrics(t.Name(), registry))
			if err != nil {
				t.Fatal(err)
			}
			controller.Update(tt.target, tt.current, 1*time.Second)

			count, err := testutil.GatherAndCount(registry, tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if count == 0 {
				t.Fatalf("expected metric %q to be registered", tt.name)
			}
			checkLabelValue(t, registry, tt.name, nameLabel, t.Name())
			if got, want := tt.value(controller.metrics), tt.want; got != want {
				t.Errorf("got %v, want: %v", got, want)
			}
		})
	}
}

func checkLabelValue(
	tb testing.TB,
	registry *prometheus.Registry,
	metric string,
	label string,
	value string,
) {
	mfs, err := registry.Gather()
	if err != nil {
		tb.Fatal(err)
	}
	for _, mf := range mfs {
		if mf.GetName() != metric {
			continue
		}
		for _, m := range mf.Metric {
			for _, l := range m.Label {
				if l.GetName() == label && l.GetValue() == value {
					return
				}
			}
		}
	}
	tb.Fatalf("expected metric %q with label %q and value: %s", metric, label, value)
}
