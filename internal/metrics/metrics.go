package metrics

import (
	"sort"
	"strings"
	"sync"
	"time"
)

const namespace = "occult"

type label struct {
	Name  string
	Value string
}

type sample struct {
	Labels []label
	Value  float64
}

type gauge struct {
	Name    string
	Help    string
	samples map[string]sample
}

var (
	mutex sync.Mutex

	success = &gauge{
		Name: namespace + "_success_bool",
		Help: "Whether unlocking was successful",
	}

	lastInvocationSeconds = &gauge{
		Name: namespace + "_last_invocation_seconds",
		Help: "Last time occult was run in a profile",
	}

	postHookSuccess = &gauge{
		Name: namespace + "_post_hook_success",
		Help: "Success of the post hooks",
	}

	gauges = []*gauge{lastInvocationSeconds, postHookSuccess, success}
)

func SetSuccess(profile string, ok bool) {
	success.set(boolToFloat(ok), label{"profile", profile})
}

func SetLastInvocation(profile string, t time.Time) {
	lastInvocationSeconds.set(float64(t.UnixMilli())/1000, label{"profile", profile})
}

func SetPostHookSuccess(profile, hook string, ok bool) {
	postHookSuccess.set(boolToFloat(ok), label{"hook", hook}, label{"profile", profile})
}

func (g *gauge) set(value float64, labels ...label) {
	mutex.Lock()
	defer mutex.Unlock()

	if g.samples == nil {
		g.samples = map[string]sample{}
	}

	keys := make([]string, 0, len(labels))
	for _, l := range labels {
		keys = append(keys, l.Name+"="+l.Value)
	}
	g.samples[strings.Join(keys, "\xff")] = sample{Labels: labels, Value: value}
}

// Samples returns the gauge's samples sorted by their labels, so the output is deterministic.
func (g *gauge) Samples() []sample {
	keys := make([]string, 0, len(g.samples))
	for key := range g.samples {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	ret := make([]sample, 0, len(keys))
	for _, key := range keys {
		ret = append(ret, g.samples[key])
	}
	return ret
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
