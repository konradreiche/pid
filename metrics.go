package pid

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	nameLabel = "name"
	termLabel = "term"
)

var (
	baseLabels = []string{
		nameLabel,
	}

	termLabels = []string{
		nameLabel,
		termLabel,
	}
)

type metrics struct {
	updatesTotal  *prometheus.CounterVec
	target        *prometheus.GaugeVec
	current       *prometheus.GaugeVec
	controlSignal *prometheus.GaugeVec
	gain          *prometheus.GaugeVec
	term          *prometheus.GaugeVec

	baseLabels prometheus.Labels
}

func newMetrics(name string, reg prometheus.Registerer) (*metrics, error) {
	metrics := &metrics{
		baseLabels: prometheus.Labels{
			nameLabel: name,
		},
	}
	var err error
	metrics.updatesTotal, err = register(reg, prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pid_updates_total",
	}, baseLabels))
	if err != nil {
		return nil, err
	}
	metrics.target, err = register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pid_target",
	}, baseLabels))
	if err != nil {
		return nil, err
	}
	metrics.current, err = register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pid_current",
	}, baseLabels))
	if err != nil {
		return nil, err
	}
	metrics.controlSignal, err = register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pid_control_signal",
	}, baseLabels))
	if err != nil {
		return nil, err
	}
	metrics.gain, err = register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pid_gain",
	}, termLabels))
	if err != nil {
		return nil, err
	}
	metrics.term, err = register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pid_term",
	}, termLabels))
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func register[C prometheus.Collector](reg prometheus.Registerer, collector C) (C, error) {
	var zero C
	if err := reg.Register(collector); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			existing, ok := are.ExistingCollector.(C)
			if !ok {
				return zero, fmt.Errorf("register: existing collector has type %T, want: %T", are.ExistingCollector, zero)
			}
			return existing, nil
		}
		return zero, err
	}
	return collector, nil
}
