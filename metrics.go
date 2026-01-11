package pid

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

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

func newMetrics(name string, reg prometheus.Registerer) (*metrics, error) {
	m := &metrics{
		labels: prometheus.Labels{
			nameLabel: name,
		},
	}
	var err error
	m.updatesTotal, err = register(reg, prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pid_updates_total",
	}, labels))
	if err != nil {
		return nil, err
	}
	m.target, err = register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pid_target",
	}, labels))
	if err != nil {
		return nil, err
	}
	m.current, err = register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pid_current",
	}, labels))
	if err != nil {
		return nil, err
	}
	m.controlSignal, err = register(reg, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pid_control_signal",
	}, labels))
	if err != nil {
		return nil, err
	}
	return m, nil
}

func register[C prometheus.Collector](reg prometheus.Registerer, collector C) (C, error) {
	var c C
	if err := reg.Register(collector); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			existing, ok := are.ExistingCollector.(C)
			if !ok {
				return c, fmt.Errorf("register: existing collector has type %T, want: %T", are.ExistingCollector, c)
			}
			return existing, nil
		}
		return c, err
	}
	return collector, nil
}
