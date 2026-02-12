package pid

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_New(t *testing.T) {
	tests := []struct {
		name  string
		opts  []Option
		value func(m *metrics) float64
		want  float64
	}{
		{
			name: "proportional_gain",
			opts: []Option{
				WithProportionalGain(3.5),
			},
			value: func(m *metrics) float64 { return testutil.ToFloat64(m.gain) },
			want:  3.5,
		},
		{
			name: "integral_gain",
			opts: []Option{
				WithIntegralGain(1.25),
			},
			value: func(m *metrics) float64 { return testutil.ToFloat64(m.gain) },
			want:  1.25,
		},
		{
			name: "derivative_gain",
			opts: []Option{
				WithDerivativeGain(0.75),
			},
			value: func(m *metrics) float64 { return testutil.ToFloat64(m.gain) },
			want:  0.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			_, err := New(
				WithPrometheusMetrics(t.Name(), registry),
				WithOptions(tt.opts...),
			)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := findLabelValue(t, registry, "pid_gain", "term", tt.name), tt.want; got != want {
				t.Errorf("got %v, want: %v", got, want)
			}
		})
	}
}

func TestMetrics_Update(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		metric string
		value  func(m *metrics, r *prometheus.Registry) float64
		want   float64
	}{
		{
			name:   "pid_updates_total",
			metric: "pid_updates_total",
			value:  func(m *metrics, r *prometheus.Registry) float64 { return testutil.ToFloat64(m.updatesTotal) },
			want:   1.0,
		},
		{
			name:   "pid_target",
			metric: "pid_target",
			value:  func(m *metrics, r *prometheus.Registry) float64 { return testutil.ToFloat64(m.target) },
			want:   5.0,
		},
		{
			name:   "pid_current",
			metric: "pid_current",
			value:  func(m *metrics, r *prometheus.Registry) float64 { return testutil.ToFloat64(m.current) },
			want:   2.0,
		},
		{
			name:   "pid_control_signal",
			metric: "pid_control_signal",
			value:  func(m *metrics, r *prometheus.Registry) float64 { return testutil.ToFloat64(m.controlSignal) },
			want:   3.0,
		},
		{
			name:   "pid_term_proportonal",
			metric: "pid_term",
			opts: []Option{
				WithProportionalGain(1.5),
			},
			value: func(m *metrics, r *prometheus.Registry) float64 {
				return findLabelValue(t, r, "pid_term", "term", "proportional")
			},
			want: 4.5,
		},
		{
			name:   "pid_term_integral",
			metric: "pid_term",
			opts: []Option{
				WithIntegralGain(2),
			},
			value: func(m *metrics, r *prometheus.Registry) float64 {
				return findLabelValue(t, r, "pid_term", "term", "integral")
			},
			want: 6.0,
		},
		{
			name:   "pid_term_derivative",
			metric: "pid_term",
			opts: []Option{
				WithDerivativeGain(3),
			},
			value: func(m *metrics, r *prometheus.Registry) float64 {
				return findLabelValue(t, r, "pid_term", "term", "derivative")
			},
			want: 9.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			controller, err := New(
				WithPrometheusMetrics(t.Name(), registry),
				WithOptions(tt.opts...),
			)
			if err != nil {
				t.Fatal(err)
			}
			controller.Update(5, 2, 1*time.Second)

			count, err := testutil.GatherAndCount(registry, tt.metric)
			if err != nil {
				t.Fatal(err)
			}
			if count == 0 {
				t.Fatalf("expected metric %q to be registered", tt.name)
			}
			if got, want := tt.value(controller.metrics, registry), tt.want; got != want {
				t.Errorf("got %v, want: %v", got, want)
			}
		})
	}
}

func TestReusePrometheusRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	a, err := New(WithPrometheusMetrics("a", registry))
	if err != nil {
		t.Fatal(err)
	}
	b, err := New(WithPrometheusMetrics("b", registry))
	if err != nil {
		t.Fatal(err)
	}

	a.Update(7, 4, 1*time.Second)
	b.Update(2, 3, 1*time.Second)

	findLabelValue(t, registry, "pid_control_signal", "name", "a")
	findLabelValue(t, registry, "pid_control_signal", "name", "b")
}

func findLabelValue(
	tb testing.TB,
	registry *prometheus.Registry,
	metric string,
	label string,
	value string,
) float64 {
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
					return m.Gauge.GetValue()
				}
			}
		}
	}
	tb.Fatalf("expected metric %q with label %q and value: %s", metric, label, value)
	return 0.0
}
