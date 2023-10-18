package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	log.Info().Msgf("Starting occult %s, commit %s", internal.BuildVersion, internal.CommitHash)
	configFilePath, err := getPreferredConfigFile()
	if err != nil {
		log.Fatal().Err(err).Msg("no config file provided")
	}

	log.Info().Msgf("Using config file %q", configFilePath)
	conf, err := config.Read(configFilePath)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read config")
	}

	if err := config.Validate(conf); err != nil {
		log.Fatal().Err(err).Msg("invalid config")
	}

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
		log.Info().Msg("Received signal, shutting down")
	case <-done:
		log.Info().Msg("Finished unlocking")
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
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if term.IsTerminal(int(os.Stdout.Fd())) {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		})
	}
}
