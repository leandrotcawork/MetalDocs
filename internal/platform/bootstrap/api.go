package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	auditdomain "metaldocs/internal/modules/audit/domain"
	auditmemory "metaldocs/internal/modules/audit/infrastructure/memory"
	auditpg "metaldocs/internal/modules/audit/infrastructure/postgres"
	authdomain "metaldocs/internal/modules/auth/domain"
	authmemory "metaldocs/internal/modules/auth/infrastructure/memory"
	authpg "metaldocs/internal/modules/auth/infrastructure/postgres"
	docdomain "metaldocs/internal/modules/documents/domain"
	memoryrepo "metaldocs/internal/modules/documents/infrastructure/memory"
	pgrepo "metaldocs/internal/modules/documents/infrastructure/postgres"
	iamdomain "metaldocs/internal/modules/iam/domain"
	iampg "metaldocs/internal/modules/iam/infrastructure/postgres"
	notificationdomain "metaldocs/internal/modules/notifications/domain"
	notificationmemory "metaldocs/internal/modules/notifications/infrastructure/memory"
	notificationpg "metaldocs/internal/modules/notifications/infrastructure/postgres"
	workflowdomain "metaldocs/internal/modules/workflow/domain"
	workflowmemory "metaldocs/internal/modules/workflow/infrastructure/memory"
	workflowpg "metaldocs/internal/modules/workflow/infrastructure/postgres"
	"metaldocs/internal/platform/authn"
	"metaldocs/internal/platform/config"
	pgdb "metaldocs/internal/platform/db/postgres"
	"metaldocs/internal/platform/messaging"
	nooppub "metaldocs/internal/platform/messaging/noop"
	outboxpg "metaldocs/internal/platform/messaging/outbox/postgres"
	"metaldocs/internal/platform/observability"
	"metaldocs/internal/platform/render/carbone"
	docgenclient "metaldocs/internal/platform/render/docgen"
	localstorage "metaldocs/internal/platform/storage/local"
	miniostorage "metaldocs/internal/platform/storage/minio"
)

type APIDependencies struct {
	DocumentsRepo     docdomain.Repository
	WorkflowApprovals workflowdomain.ApprovalRepository
	AttachmentStore   docdomain.AttachmentStore
	CarboneClient     *carbone.Client
	CarboneTemplates  *carbone.TemplateRegistry
	RoleProvider      iamdomain.RoleProvider
	RoleAdminRepo     iamdomain.RoleAdminRepository
	AuthRepo          authdomain.Repository
	NotificationsRepo notificationdomain.Repository
	AuditWriter       auditdomain.Writer
	AuditReader       auditdomain.Reader
	Publisher         messaging.Publisher
	DocgenClient      *docgenclient.Client
	StatusProvider    observability.RuntimeStatusProvider
	Cleanup           func()
}

type bucketEnsurer interface {
	EnsureBucket(ctx context.Context) error
}

func BuildAPIDependencies(ctx context.Context, repoMode string, attachmentsCfg config.AttachmentsConfig, carboneCfg config.CarboneConfig) (APIDependencies, error) {
	carboneClient := carbone.NewClient(carboneCfg)
	carboneRegistry, err := carbone.BootstrapTemplates(ctx, carboneClient, carboneCfg, nil)
	if err != nil {
		log.Printf("carbone bootstrap degraded: %v", err)
	}
	docgenClient := docgenclient.NewClient(config.LoadDocgenConfig())
	carboneCheck := observability.DependencyCheck{
		Name: "carbone",
		Check: func(ctx context.Context) (observability.DependencyCheckResult, error) {
			if !carboneCfg.Enabled {
				return observability.DependencyCheckResult{
					Status: "skipped",
					Detail: "carbone disabled",
				}, nil
			}
			if carboneClient == nil {
				return observability.DependencyCheckResult{}, fmt.Errorf("carbone client not configured")
			}
			if err := carboneClient.Ping(ctx, "health"); err != nil {
				return observability.DependencyCheckResult{}, err
			}
			meta := map[string]any{}
			if carboneRegistry != nil {
				meta["templates"] = carboneRegistry.Count()
			}
			return observability.DependencyCheckResult{Status: "up", Meta: meta}, nil
		},
	}

	switch repoMode {
	case config.RepositoryPostgres:
		pgCfg, err := config.LoadPostgresConfig()
		if err != nil {
			return APIDependencies{}, fmt.Errorf("load postgres config: %w", err)
		}
		db, err := pgdb.Open(ctx, pgCfg.DSN)
		if err != nil {
			return APIDependencies{}, fmt.Errorf("open postgres: %w", err)
		}

		store, err := buildAttachmentStore(ctx, attachmentsCfg)
		if err != nil {
			_ = closeDB(db)
			return APIDependencies{}, err
		}

		authRepo := authpg.NewRepository(db)
		return APIDependencies{
			DocumentsRepo:     pgrepo.NewRepository(db),
			WorkflowApprovals: workflowpg.NewApprovalRepository(db),
			AttachmentStore:   store,
			CarboneClient:     carboneClient,
			CarboneTemplates:  carboneRegistry,
			RoleProvider:      iampg.NewRoleProvider(db),
			RoleAdminRepo:     iampg.NewRoleAdminRepository(db),
			AuthRepo:          authRepo,
			NotificationsRepo: notificationpg.NewRepository(db),
			AuditWriter:       auditpg.NewWriter(db),
			AuditReader:       auditpg.NewWriter(db),
			Publisher:         outboxpg.NewPublisher(db),
			DocgenClient:      docgenClient,
			StatusProvider:    observability.NewPostgresRuntimeStatusProvider(db, repoMode, attachmentsCfg.Provider, authn.Enabled(), carboneCheck),
			Cleanup:           func() { _ = closeDB(db) },
		}, nil
	default:
		roles := authn.DevRoleMap()
		store, err := buildAttachmentStore(ctx, attachmentsCfg)
		if err != nil {
			return APIDependencies{}, err
		}
		authRepo := authmemory.NewRepository()
		for userID, userRoles := range roles {
			if err := authRepo.UpsertUserAndAssignRole(ctx, userID, userID, userRoles[0], "bootstrap"); err != nil {
				return APIDependencies{}, err
			}
			for _, role := range userRoles[1:] {
				if err := authRepo.UpsertUserAndAssignRole(ctx, userID, userID, role, "bootstrap"); err != nil {
					return APIDependencies{}, err
				}
			}
		}
		auditStore := auditmemory.NewWriter()
		return APIDependencies{
			DocumentsRepo:     memoryrepo.NewRepository(),
			WorkflowApprovals: workflowmemory.NewApprovalRepository(),
			AttachmentStore:   store,
			CarboneClient:     carboneClient,
			CarboneTemplates:  carboneRegistry,
			RoleProvider:      authRepo,
			RoleAdminRepo:     authRepo,
			AuthRepo:          authRepo,
			NotificationsRepo: notificationmemory.NewRepository(),
			AuditWriter:       auditStore,
			AuditReader:       auditStore,
			Publisher:         nooppub.NewPublisher(),
			DocgenClient:      docgenClient,
			StatusProvider:    observability.NewStaticRuntimeStatusProvider(repoMode, attachmentsCfg.Provider, authn.Enabled(), carboneCheck),
			Cleanup:           func() {},
		}, nil
	}
}

func buildAttachmentStore(ctx context.Context, cfg config.AttachmentsConfig) (docdomain.AttachmentStore, error) {
	switch cfg.Provider {
	case config.StorageProviderMemory:
		return memoryrepo.NewAttachmentStore(), nil
	case config.StorageProviderMinIO:
		store, err := miniostorage.NewStore(cfg)
		if err != nil {
			return nil, err
		}
		if err := store.EnsureBucket(ctx); err != nil {
			return nil, err
		}
		return store, nil
	default:
		return localstorage.NewStore(cfg.RootPath), nil
	}
}

func closeDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}
