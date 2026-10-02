package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/soerenschneider/occult/v2/internal"
	"github.com/soerenschneider/occult/v2/internal/config"
	"github.com/soerenschneider/occult/v2/internal/metrics"
	"go.uber.org/multierr"
	"golang.org/x/term"
)

var (
	defaultConfigFileLocations = []string{"occult.yaml", "~/.occult.yaml", "/etc/occult.yaml"}
	ErrNoConfigFile            = fmt.Errorf("no config file defined and no implicit config files found at %s", strings.Join(defaultConfigFileLocations, ", "))
)

var (
	configFile   string
	printVersion bool
	debug        bool
)

func main() {
	parseFlags()
	if printVersion {
		fmt.Println(internal.BuildVersion)
		os.Exit(0)
	}

	initLogging()

	slog.Info("Starting occult", "version", internal.BuildVersion, "commit", internal.CommitHash)
	configFilePath, err := getPreferredConfigFile()
	dieOnError(err, "no config file provided")

	slog.Info("Using config file", "path", configFilePath)
	conf, err := config.Read(configFilePath)
	dieOnError(err, "could not read config")

	err = config.Validate(conf)
	dieOnError(err, "invalid config")

	deps := buildDeps(*conf)

	err = run(deps, *conf)
	if len(conf.MetricsPath) > 0 {
		if metricWriteErr := metrics.WriteMetrics(conf.MetricsPath); metricWriteErr != nil {
			err = multierr.Append(err, metricWriteErr)
		}
	}
	dieOnError(err, "errors while running occult")
}

func run(deps *dependencies, conf config.OccultConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	done := make(chan bool)
	errChan := make(chan error)
	go func() {
		err := deps.occult.Run(ctx, conf, wg)
		if err != nil {
			errChan <- fmt.Errorf("one or more unlock requests failed: %w", err)
		}
		done <- true
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	var err error
	select {
	case <-sig:
		slog.Info("Received signal, shutting down")
	case <-done:
		slog.Info("Finished unlocking")
	case e := <-errChan:
		err = e
	}

	cancel()
	wg.Wait()

	return err
}

func parseFlags() {
	flag.StringVar(&configFile, "config", "", fmt.Sprintf("Path to the config file (default %v)", defaultConfigFileLocations))
	flag.BoolVar(&printVersion, "version", false, "Print printVersion and exit")
	flag.BoolVar(&debug, "debug", false, "Print debug information")
	flag.Parse()
}

func getPreferredConfigFile() (string, error) {
	if len(configFile) > 0 {
		return internal.ExpandTilde(configFile), nil
	}

	for _, path := range defaultConfigFileLocations {
		path = internal.ExpandTilde(path)
		_, err := os.Stat(path)
		if err == nil {
			return path, nil
		}
	}

	return "", ErrNoConfigFile
}

func initLogging() {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if debug {
		opts.Level = slog.LevelDebug
	}

	var handler slog.Handler = slog.NewJSONHandler(os.Stderr, opts)
	if term.IsTerminal(int(os.Stdout.Fd())) {
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.String(slog.TimeKey, a.Value.Time().Format(time.TimeOnly))
			}
			return a
		}
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}
