// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	internalVault "github.com/bank-vaults/bank-vaults/internal/vault"
)

const prometheusNS = "vault"

var (
	initializedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNS, "sys", "initialized"),
		"Is the Vault node initialized.",
		nil, nil,
	)
	sealedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNS, "sys", "sealed"),
		"Is the Vault node sealed.",
		nil, nil,
	)
	leaderDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNS, "sys", "leader"),
		"Is the Vault node the leader.",
		nil, nil,
	)
	successfulConfigurationsCount float64
	successfulConfigurationsDesc  = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNS, "config", "successful"),
		"Number of successful configurations files applied",
		nil, nil,
	)
	failedConfigurationsCount float64
	failedConfigurationsDesc  = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNS, "config", "failed"),
		"Number of configurations files applied that failed",
		nil, nil,
	)
)

type prometheusExporter struct {
	Vault internalVault.Vault
	Mode  string
}

func (e *prometheusExporter) Describe(ch chan<- *prometheus.Desc) {
	switch e.Mode {
	case "unseal":
		ch <- initializedDesc
		ch <- sealedDesc
		ch <- leaderDesc
	case "configure":
		ch <- successfulConfigurationsDesc
		ch <- failedConfigurationsDesc
	}
}

func (e *prometheusExporter) Collect(ch chan<- prometheus.Metric) {
	switch e.Mode {
	case "unseal":
		sealed, err := e.Vault.Sealed()
		if err != nil {
			slog.Error(fmt.Sprintf("error checking if vault is sealed: %s", err.Error()))
			return
		}

		leader, err := e.Vault.Leader()
		if err != nil {
			slog.Error(fmt.Sprintf("error checking if vault is leader: %s", err.Error()))
			return
		}

		ch <- prometheus.MustNewConstMetric(
			initializedDesc, prometheus.GaugeValue, bToF(true),
		)
		ch <- prometheus.MustNewConstMetric(
			sealedDesc, prometheus.GaugeValue, bToF(sealed),
		)
		ch <- prometheus.MustNewConstMetric(
			leaderDesc, prometheus.GaugeValue, bToF(leader),
		)
	case "configure":
		ch <- prometheus.MustNewConstMetric(
			successfulConfigurationsDesc, prometheus.GaugeValue, successfulConfigurationsCount,
		)
		ch <- prometheus.MustNewConstMetric(
			failedConfigurationsDesc, prometheus.GaugeValue, failedConfigurationsCount,
		)
	}
}

func (e prometheusExporter) Run() error {
	slog.Info(fmt.Sprintf("vault metrics exporter enabled: %s%s", ":9091", "/metrics"))
	prometheus.MustRegister(&e)
	http.DefaultServeMux.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":9091", http.DefaultServeMux) //nolint:gosec
}

func bToF(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
