package metrics

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

var textfileTemplate = template.Must(template.New("textfile").Funcs(template.FuncMap{
	"labels": formatLabels,
	"value":  formatValue,
}).Parse(`{{- range . }}{{ $gauge := . }}{{ with .Samples -}}
# HELP {{ $gauge.Name }} {{ $gauge.Help }}
# TYPE {{ $gauge.Name }} gauge
{{ range . }}{{ $gauge.Name }}{{ labels .Labels }} {{ value .Value }}
{{ end }}{{ end }}{{ end -}}
`))

var labelValueEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)

func formatLabels(labels []label) string {
	if len(labels) == 0 {
		return ""
	}

	formatted := make([]string, 0, len(labels))
	for _, l := range labels {
		formatted = append(formatted, l.Name+`="`+labelValueEscaper.Replace(l.Value)+`"`)
	}
	return "{" + strings.Join(formatted, ",") + "}"
}

func formatValue(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func WriteMetrics(dir string) error {
	path := filepath.Join(dir, "occult.prom")

	slog.Info("Dumping metrics", "path", path)
	metrics, err := dumpMetrics()
	if err != nil {
		return err
	}

	return writeFileAtomic(path, metrics)
}

func dumpMetrics() ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()

	var buf bytes.Buffer
	if err := textfileTemplate.Execute(&buf, gauges); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeFileAtomic writes data to a temporary file in the same directory and renames it afterwards, so readers such as
// node_exporter never see a partially written file.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	// no-op after a successful rename
	defer os.Remove(tmp.Name()) //nolint:errcheck

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
