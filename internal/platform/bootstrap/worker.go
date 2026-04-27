package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pgrepo "metaldocs/internal/modules/documents/infrastructure/postgres"
	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationpg "metaldocs/internal/modules/notifications/infrastructure/postgres"
	"metaldocs/internal/platform/config"
	pgdb "metaldocs/internal/platform/db/postgres"
	"metaldocs/internal/platform/messaging"
	outboxpg "metaldocs/internal/platform/messaging/outbox/postgres"
	"metaldocs/internal/platform/servicebus"
)

type WorkerDependencies struct {
	Consumer         messaging.Consumer
	NotificationsSvc *notificationapp.Service
	DocgenV2Client   *servicebus.DocgenV2Client
	SQLDB            *sql.DB
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

	docgenV2Cfg := config.LoadDocgenV2Config()
	var docgenV2Client *servicebus.DocgenV2Client
	if docgenV2Cfg.Enabled {
		docgenV2Client = servicebus.NewDocgenV2Client(
			docgenV2Cfg.APIURL,
			docgenV2Cfg.ServiceToken,
			time.Duration(docgenV2Cfg.RequestTimeoutSeconds)*time.Second,
		)
	}

	docRepo := pgrepo.NewRepository(db)
	notificationsRepo := notificationpg.NewRepository(db)
	consumer := outboxpg.NewConsumer(db, time.Duration(workerCfg.RetryBaseSeconds)*time.Second)
	notificationsSvc := notificationapp.NewService(notificationsRepo, docRepo, nil)

	return WorkerDependencies{
		Consumer:         consumer,
		NotificationsSvc: notificationsSvc,
		DocgenV2Client:   docgenV2Client,
		SQLDB:            db,
		Cleanup:          func() { _ = closeDB(db) },
	}, nil
}
