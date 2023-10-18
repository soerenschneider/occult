package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/occult/v2/internal/config"
	"github.com/soerenschneider/occult/v2/internal/metrics"
	"github.com/soerenschneider/occult/v2/internal/precondition"
	"go.uber.org/multierr"
)

// Vault describes all the operations needed to receive the secret data
type Vault interface {
	ReadKv2(ctx context.Context, path string) (map[string]any, error)
	ReadTransitSecret(ctx context.Context, path string, ciphertext string) (map[string]any, error)
}

type Precondition interface {
	ShouldPerformUnlock(ctx context.Context) bool
}

type Occult struct {
	vault Vault
	conf  config.OccultConfig
}

func NewOccult(vault Vault, conf config.OccultConfig) (*Occult, error) {
	o := &Occult{
		vault: vault,
		conf:  conf,
	}

	return o, nil
}

func (o *Occult) Run(ctx context.Context, conf config.OccultConfig, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()

	var errs error
	for _, req := range conf.UnlockRequests {
		func() {
			ctx, cancel := context.WithTimeout(context.WithValue(ctx, "profile", req.Profile), time.Minute*1)
			defer cancel()

			if err := o.performUnlockRequest(ctx, req); err != nil {
				errs = multierr.Append(errs, err)
			}
		}()
	}

	return errs
}

func (o *Occult) performUnlockRequest(ctx context.Context, req config.UnlockConfig) error {
	if req.Precondition != nil {
		precondition, err := buildPreconditionImpl(req.Precondition)
		if err != nil {
			return fmt.Errorf("could not build precondition for profile %s: %w", req.Profile, err)
		}

		log.Info().Msgf("Evaluating precondition")
		if !precondition.ShouldPerformUnlock(ctx) {
			log.Info().Msgf("Precondition indicates no unlocking necessary for profile %s", req.Profile)
			return nil
		}
	}

	if err := o.unlock(ctx, req); err != nil {
		return fmt.Errorf("could not run profile %q: %w", req.Profile, err)
	}
	return nil
}

func buildPreconditionImpl(conf *config.PreconditionConfigContainer) (Precondition, error) {
	switch conf.PreconditionConfig.GetType() {
	case config.PreconditionCmdType:
		cmdConf := conf.PreconditionConfig.(*config.CmdPreconditionConfig)
		return precondition.NewCmd(*cmdConf)
	case config.PreconditionPathType:
		pathConf := conf.PreconditionConfig.(*config.PathPreconditionConfig)
		return precondition.NewPathPrecondition(*pathConf)
	default:
		return nil, errors.New("no precondition to build")
	}
}

func (o *Occult) unlock(ctx context.Context, conf config.UnlockConfig) error {
	secret, err := o.readSecret(ctx, conf)
	if err != nil {
		return err
	}

	payload, ok := secret[conf.Accessor]
	if !ok {
		return fmt.Errorf("%q not found in secret data", conf.Accessor)
	}

	payloadStr, ok := payload.(string)
	if !ok {
		return errors.New("can not convert secret to string")
	}

	if err := runUnlockCommand(ctx, conf.Command, payloadStr); err != nil {
		return err
	}

	if len(conf.PostHooks) > 0 {
		log.Info().Msgf("Running %d post-hooks", len(conf.PostHooks))
		return runPosthooks(ctx, conf.PostHooks, conf.PostHooksStopOnError)
	}

	return nil
}

func (o *Occult) readSecret(ctx context.Context, conf config.UnlockConfig) (map[string]any, error) {
	switch conf.SecretType {
	case config.Kv2SecretType:
		return o.vault.ReadKv2(ctx, conf.SecretPath)
	case config.TransitSecretType:
		var ciphertext = conf.CipherTextData
		if !isBase64Encoded(ciphertext) {
			ciphertext = base64.StdEncoding.EncodeToString([]byte(ciphertext))
		}

		return o.vault.ReadTransitSecret(ctx, conf.SecretPath, ciphertext)
	}

	return nil, errors.New("unknown implementation")
}

func runUnlockCommand(ctx context.Context, c string, payload string) error {
	cmdWithArgs := strings.Split(c, " ")
	log.Info().Msgf("Running command %q", cmdWithArgs[0])

	cmd := exec.CommandContext(ctx, cmdWithArgs[0], cmdWithArgs[1:]...) // #nosec: G204
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(payload)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", stderr.String(), err)
	}

	return nil
}

func runPosthooks(ctx context.Context, cmds []string, stopOnError bool) error {
	var errs error

	for _, c := range cmds {
		cmdWithArgs := strings.Split(c, " ")
		log.Info().Msgf("Running post pook command %v", cmdWithArgs)
		cmd := exec.CommandContext(ctx, cmdWithArgs[0], cmdWithArgs[1:]...) // #nosec: G204
		profile := safeCtxValue(ctx, "profile", "UNKNOWN")
		if err := cmd.Run(); err != nil {
			metrics.PostHookSuccess.WithLabelValues(profile, cmdWithArgs[0]).Set(0)
			errs = multierr.Append(errs, err)

			if stopOnError {
				return errs
			}
		}
		metrics.PostHookSuccess.WithLabelValues(profile, cmdWithArgs[0]).Set(1)
	}

	return errs
}

func isBase64Encoded(input string) bool {
	_, err := base64.StdEncoding.DecodeString(input)
	return err == nil
}

func safeCtxValue(ctx context.Context, key, def string) string {
	val := ctx.Value(key)
	if val == nil {
		return def
	}

	valStr, ok := val.(string)
	if ok {
		return valStr
	}

	return def
}
