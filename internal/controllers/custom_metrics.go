/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// these metrics are introduced for learning purposes, these have almost no real value

package controllers

import (
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

// RegisterMetrics registers the package's Prometheus counters with the supplied
// registry. Call this once from SetupWithManager instead of relying on init()
// so the controllers package can be imported in unit tests without side effects
// on the global Prometheus registry.
func RegisterMetrics(registry prometheus.Registerer) {
	registry.MustRegister(
		sopsSecretsReconciliations,
		sopsSecretsReconciliationFailures,
		sopsSecretsReconciliationsSuspended,
	)
}
