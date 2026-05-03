package heartbeat

import (
	"context"
	"log/slog"
	"os"
	"time"
)

const heartbeatFile = "/tmp/heartbeat"

func StartHeartbeat(ctx context.Context, cancel context.CancelFunc) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	err := os.MkdirAll("/tmp", 0755)
	if err != nil {
		logger.Error("Failed to create /tmp folder, exiting", "error", err)
		os.Exit(1)
	}
	f, err := os.OpenFile(heartbeatFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("Could not create or open heartbeat file, exiting", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logger.Info("Error happened during closing the hearbeat file", "error", err)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			logger.Info("Context stopped heartbeat")
			return
		case <-ticker.C:
			now := time.Now()
			err = os.WriteFile(heartbeatFile, []byte(now.String()), 0644)
			if err != nil {
				logger.Error("Failed to write in heartbeat file", "error", err)
				cancel()
				return
			}
		}
	}
}
