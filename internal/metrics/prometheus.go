package metrics

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog/log"
)

func WriteMetrics(dir string) error {
	path := filepath.Join(dir, "occult.prom")

	log.Info().Msgf("Dumping metrics to %s", path)
	metrics, err := dumpMetrics()
	if err != nil {
		return err
	}

	return writeFileAtomic(path, []byte(metrics))
}

// writeFileAtomic writes data to a temporary file in the same directory and renames it afterwards, so readers such as
// node_exporter never see a partially written file.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	// no-op after a successful rename
	defer func() {
		_ = os.Remove(tmp.Name())
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// CreateTemp uses 0600, but node_exporter usually runs as a different user
	if err := os.Chmod(tmp.Name(), 0644); err != nil { // #nosec: G302
		return err
	}

	return os.Rename(tmp.Name(), path)
}

func dumpMetrics() (string, error) {
	var buf = &bytes.Buffer{}
	fmt := expfmt.NewFormat(expfmt.TypeTextPlain)
	enc := expfmt.NewEncoder(buf, fmt)

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
