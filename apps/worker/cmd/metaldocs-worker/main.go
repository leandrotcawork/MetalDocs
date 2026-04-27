package main

import (
	"context"
	"log"
	"time"

	docrepo "metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/platform/bootstrap"
	"metaldocs/internal/platform/config"
	workerapp "metaldocs/internal/platform/worker"
)

func main() {
	workerCfg, err := config.LoadWorkerConfig()
	if err != nil {
		log.Fatalf("invalid worker config: %v", err)
	}
	deps, err := bootstrap.BuildWorkerDependencies(context.Background(), workerCfg)
	if err != nil {
		log.Fatalf("build worker dependencies: %v", err)
	}
	defer deps.Cleanup()

	workerSvc := workerapp.NewService(deps.Consumer, deps.NotificationsSvc, workerCfg)
	if deps.DocgenV2Client != nil && deps.SQLDB != nil {
		snapRepo := docrepo.NewSnapshotRepository(deps.SQLDB)
		pdfRunner := workerapp.NewPDFJobRunner(deps.DocgenV2Client, snapRepo)
		workerSvc = workerSvc.WithPDFRunner(pdfRunner)
	}

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
	log.Printf("MetalDocs Worker running (poll_interval_s=%d batch_size=%d review_reminder_days=%d max_attempts=%d retry_base_seconds=%d retry_max_seconds=%d)",
		workerCfg.PollIntervalSeconds, workerCfg.BatchSize, workerCfg.ReviewReminderDays, workerCfg.MaxAttempts, workerCfg.RetryBaseSeconds, workerCfg.RetryMaxSeconds)

	for {
		run()
		<-ticker.C
	}
}
