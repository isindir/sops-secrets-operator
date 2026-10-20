/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// these metrics are introduced for learning purposes, these have almost no real value

package controllers

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	sopsSecretsReconciliations = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sopssecrets_reconcilation_successes_total",
			Help: "Number of SopsSecrets reconcilations",
		},
	)

	sopsSecretsReconciliationFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sopssecrets_reconcilation_failures_total",
			Help: "Number of SopsSecrets reconcolation failures",
		},
	)

	sopsSecretsReconciliationsSuspended = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sopssecrets_reconcilation_suspends_total",
			Help: "Number of SopsSecrets reconcilations suspends",
		},
	)
)

// registerOnce ensures the Prometheus counters are registered exactly once per
// process, no matter how many times RegisterMetrics is called.
var registerOnce sync.Once

// RegisterMetrics registers the package's Prometheus counters with the supplied
// registry. Call this from SetupWithManager instead of relying on init() so the
// controllers package can be imported in unit tests without side effects on the
// global Prometheus registry.
//
// The registration is idempotent: the first call registers the counters — and
// still panics on a genuine name conflict, via MustRegister — while every later
// call (e.g. a second SetupWithManager) is a no-op. This preserves the old
// init() "safe to call repeatedly" guarantee without the global side effect.
func RegisterMetrics(registry prometheus.Registerer) {
	registerOnce.Do(func() {
		registry.MustRegister(
			sopsSecretsReconciliations,
			sopsSecretsReconciliationFailures,
			sopsSecretsReconciliationsSuspended,
		)
	})
}
