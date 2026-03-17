package main

import (
	"context"
	"log"
	"time"

	pgrepo "metaldocs/internal/modules/documents/infrastructure/postgres"
	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationpg "metaldocs/internal/modules/notifications/infrastructure/postgres"
	"metaldocs/internal/platform/config"
	pgdb "metaldocs/internal/platform/db/postgres"
	outboxpg "metaldocs/internal/platform/messaging/outbox/postgres"
	workerapp "metaldocs/internal/platform/worker"
)

func main() {
	workerCfg, err := config.LoadWorkerConfig()
	if err != nil {
		log.Fatalf("invalid worker config: %v", err)
	}
	pgCfg, err := config.LoadPostgresConfig()
	if err != nil {
		log.Fatalf("load postgres config: %v", err)
	}

	db, err := pgdb.Open(context.Background(), pgCfg.DSN)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	docRepo := pgrepo.NewRepository(db)
	notificationsRepo := notificationpg.NewRepository(db)
	consumer := outboxpg.NewConsumer(db)
	notificationsSvc := notificationapp.NewService(notificationsRepo, docRepo, nil)
	workerSvc := workerapp.NewService(consumer, notificationsSvc, workerCfg.ReviewReminderDays)

	run := func() {
		if err := workerSvc.RunOnce(context.Background(), workerCfg.BatchSize); err != nil {
			log.Printf("worker run failed: %v", err)
			return
		}
		log.Printf("worker batch completed")
	}

	if workerCfg.RunOnce {
		run()
		return
	}

	ticker := time.NewTicker(time.Duration(workerCfg.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()
	log.Printf("MetalDocs Worker running (poll_interval_s=%d batch_size=%d review_reminder_days=%d)", workerCfg.PollIntervalSeconds, workerCfg.BatchSize, workerCfg.ReviewReminderDays)

	for {
		run()
		<-ticker.C
	}
}
