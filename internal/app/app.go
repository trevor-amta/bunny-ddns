package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/trevorspencer/bunny-dynamic-dns/internal/bunny"
	"github.com/trevorspencer/bunny-dynamic-dns/internal/config"
	"github.com/trevorspencer/bunny-dynamic-dns/internal/ip"
)

// Run starts the polling loop and blocks until context cancellation.
func Run(ctx context.Context, cfg *config.Config, out io.Writer) error {
	logger := log.New(out, "", log.LstdFlags|log.LUTC)

	ipProvider := ip.NewProvider(cfg.IPProviders, cfg.UserAgent)
	bunnyClient := bunny.NewClient(cfg.ZoneID, cfg.APIKey, cfg.UserAgent)

	logger.Printf("starting bunny dynamic dns: poll_interval=%s endpoints=%d records=%d",
		cfg.PollInterval, len(cfg.IPProviders), len(cfg.Records))

	lastIP := ""

	if err := syncOnce(ctx, logger, ipProvider, bunnyClient, cfg, &lastIP); err != nil {
		logger.Printf("initial sync error: %v", err)
		// keep running—maybe transient network issue
	}

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Println("shutdown signal received, exiting")
			return nil
		case <-ticker.C:
			if err := syncOnce(ctx, logger, ipProvider, bunnyClient, cfg, &lastIP); err != nil {
				logger.Printf("sync error: %v", err)
			}
		}
	}
}

func syncOnce(
	ctx context.Context,
	logger *log.Logger,
	ipProvider *ip.Provider,
	bunnyClient *bunny.Client,
	cfg *config.Config,
	lastIP *string,
) error {
	currentIP, err := ipProvider.CurrentIP(ctx)
	if err != nil {
		return err
	}

	if *lastIP == currentIP && currentIP != "" {
		logger.Printf("wan ip unchanged (%s), verifying bunny dns records", currentIP)
	} else {
		logger.Printf("detected ip change: from=%s to=%s", prevValue(*lastIP), currentIP)
	}

	allSynced := true

	for i := range cfg.Records {
		record := &cfg.Records[i]
		remote, err := bunnyClient.GetRecord(ctx, record.ID)
		if err != nil {
			if errors.Is(err, bunny.ErrNotFound) {
				return fmt.Errorf("record id=%d name=%s not found in bunny; confirm the id matches the zone", record.ID, record.Name)
			}

			return fmt.Errorf("fetch record id=%d name=%s: %w", record.ID, record.Name, err)
		}

		if !strings.EqualFold(remote.Type, record.Type) {
			logger.Printf("warning: record id=%d name=%s type mismatch config=%s remote=%s",
				record.ID, record.Name, record.Type, remote.Type)
		}

		remoteValue := strings.TrimSpace(remote.Value)
		if remoteValue == currentIP {
			logger.Printf("record id=%d name=%s already set to %s", record.ID, record.Name, currentIP)
			continue
		}

		allSynced = false

		if err := bunnyClient.UpdateRecord(ctx, *record, currentIP); err != nil {
			return fmt.Errorf("update record id=%d name=%s: %w", record.ID, record.Name, err)
		}

		logger.Printf("updated record id=%d name=%s from=%s to=%s", record.ID, record.Name, remoteValue, currentIP)
	}

	if allSynced {
		logger.Printf("all records already publishing %s", currentIP)
	}

	*lastIP = currentIP

	return nil
}

func prevValue(ip string) string {
	if ip == "" {
		return "(none)"
	}
	return ip
}
