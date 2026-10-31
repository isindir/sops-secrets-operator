/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// these metrics are introduced for learning purposes, these have almost no real value

package controllers

import (
	"errors"

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
// registry. Call this from SetupWithManager instead of relying on init() so the
// controllers package can be imported in unit tests without side effects on the
// global Prometheus registry.
//
// Registration is idempotent per registry: calling it again with the same
// registry is a no-op, while each new registry gets its own registration. It
// panics if a different collector already owns one of the metric names.
func RegisterMetrics(registry prometheus.Registerer) {
	for _, c := range []prometheus.Collector{
		sopsSecretsReconciliations,
		sopsSecretsReconciliationFailures,
		sopsSecretsReconciliationsSuspended,
	} {
		if err := registry.Register(c); err != nil {
			var alreadyRegistered prometheus.AlreadyRegisteredError
			if !errors.As(err, &alreadyRegistered) || alreadyRegistered.ExistingCollector != c {
				panic(err)
			}
		}
	}
}
