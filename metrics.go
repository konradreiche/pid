package pid

import "github.com/prometheus/client_golang/prometheus"

const nameLabel = "name"

var labels = []string{
	nameLabel,
}

type metrics struct {
	updatesTotal  *prometheus.CounterVec
	target        *prometheus.GaugeVec
	current       *prometheus.GaugeVec
	controlSignal *prometheus.GaugeVec

	labels prometheus.Labels
}

func newMetrics(name string, registerer prometheus.Registerer) (*metrics, error) {
	metrics := &metrics{
		updatesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pid_updates_total",
		}, labels),
		target: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "pid_target",
		}, labels),
		current: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "pid_current",
		}, labels),
		controlSignal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "pid_control_signal",
		}, labels),
		labels: prometheus.Labels{
			nameLabel: name,
		},
	}

	for _, collector := range []prometheus.Collector{
		metrics.updatesTotal,
		metrics.target,
		metrics.current,
		metrics.controlSignal,
	} {
		if err := registerer.Register(collector); err != nil {
			return nil, err
		}
	}
	return metrics, nil
}
