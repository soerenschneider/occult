package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "occult"

var (
	Success = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "success_bool",
		Help:      "Whether unlocking was successful",
	}, []string{"profile"})

	LastInvocationSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "last_invocation_seconds",
		Help:      "Last time occult was run in a profile",
	}, []string{"profile"})

	PostHookSuccess = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "post_hook_success",
		Help:      "Success of the post hooks",
	}, []string{"profile", "hook"})
)
