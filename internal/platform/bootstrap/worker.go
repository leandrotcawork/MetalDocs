package bootstrap

import (
	"context"
	"fmt"
	"time"

	pgrepo "metaldocs/internal/modules/documents/infrastructure/postgres"
	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationpg "metaldocs/internal/modules/notifications/infrastructure/postgres"
	"metaldocs/internal/platform/config"
	pgdb "metaldocs/internal/platform/db/postgres"
	"metaldocs/internal/platform/messaging"
	outboxpg "metaldocs/internal/platform/messaging/outbox/postgres"
)

type WorkerDependencies struct {
	Consumer         messaging.Consumer
	NotificationsSvc *notificationapp.Service
	Cleanup          func()
}

func BuildWorkerDependencies(ctx context.Context, workerCfg config.WorkerConfig) (WorkerDependencies, error) {
	pgCfg, err := config.LoadPostgresConfig()
	if err != nil {
		return WorkerDependencies{}, fmt.Errorf("load postgres config: %w", err)
	}
	db, err := pgdb.Open(ctx, pgCfg.DSN)
	if err != nil {
		return WorkerDependencies{}, fmt.Errorf("open postgres: %w", err)
	}

	docRepo := pgrepo.NewRepository(db)
	notificationsRepo := notificationpg.NewRepository(db)
	consumer := outboxpg.NewConsumer(db, time.Duration(workerCfg.RetryBaseSeconds)*time.Second)
	notificationsSvc := notificationapp.NewService(notificationsRepo, docRepo, nil)

	return WorkerDependencies{
		Consumer:         consumer,
		NotificationsSvc: notificationsSvc,
		Cleanup:          func() { _ = closeDB(db) },
	}, nil
}
