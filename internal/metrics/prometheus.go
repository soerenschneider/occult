package metrics

import (
	"bytes"
	"net/url"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog/log"
)

func WriteMetrics(path string) error {
	path, err := url.JoinPath(path, "occult.prom")
	if err != nil {
		return err
	}

	log.Info().Msgf("Dumping metrics to %s", path)
	metrics, err := dumpMetrics()
	if err != nil {
		return err
	}

	err = os.WriteFile(path, []byte(metrics), 0644) // #nosec: G306
	return err
}

func dumpMetrics() (string, error) {
	var buf = &bytes.Buffer{}
	enc := expfmt.NewEncoder(buf, expfmt.FmtText)

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return "", err
	}

	for _, f := range families {
		// Writing these metrics will cause a duplication error with other tools writing the same metrics
		if !strings.HasPrefix(f.GetName(), "go_") {
			if err := enc.Encode(f); err != nil {
				log.Info().Msgf("could not encode metric: %s", err.Error())
			}
		}
	}

	return buf.String(), nil
}
