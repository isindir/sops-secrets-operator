/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package controllers

import (
	"slices"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var wantMetricNames = []string{
	"sopssecrets_reconcilation_failures_total",
	"sopssecrets_reconcilation_successes_total",
	"sopssecrets_reconcilation_suspends_total",
}

func gatheredMetricNames(t *testing.T, reg *prometheus.Registry) []string {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}
	names := make([]string, 0, len(families))
	for _, f := range families {
		names = append(names, f.GetName())
	}
	return names
}

func TestRegisterMetricsExposesCountersOnEveryRegistry(t *testing.T) {
	for _, name := range []string{"first registry", "second registry"} {
		t.Run(name, func(t *testing.T) {
			reg := prometheus.NewRegistry()

			RegisterMetrics(reg)

			if got := gatheredMetricNames(t, reg); !slices.Equal(got, wantMetricNames) {
				t.Errorf("registry exposes %v, want %v", got, wantMetricNames)
			}
		})
	}
}

func TestRegisterMetricsTwiceOnSameRegistryDoesNotPanic(t *testing.T) {
	reg := prometheus.NewRegistry()
	RegisterMetrics(reg)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second RegisterMetrics on the same registry panicked: %v", r)
		}
	}()
	RegisterMetrics(reg)

	if got := gatheredMetricNames(t, reg); !slices.Equal(got, wantMetricNames) {
		t.Errorf("registry exposes %v, want %v", got, wantMetricNames)
	}
}

func TestRegisterMetricsPanicsWhenAnotherCollectorOwnsTheName(t *testing.T) {
	reg := prometheus.NewRegistry()
	impostor := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sopssecrets_reconcilation_successes_total",
		Help: "Number of SopsSecrets reconcilations",
	})
	reg.MustRegister(impostor)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("RegisterMetrics must panic when a different collector already owns one of its metric names")
		}
	}()
	RegisterMetrics(reg)
}
