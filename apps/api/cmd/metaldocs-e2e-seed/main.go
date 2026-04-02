package main

import (
	"context"
	"log"
	"os"
	"strings"

	authapp "metaldocs/internal/modules/auth/application"
	authdomain "metaldocs/internal/modules/auth/domain"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/platform/authn"
	"metaldocs/internal/platform/bootstrap"
	"metaldocs/internal/platform/config"
)

type seedConfig struct {
	UserID      string
	Username    string
	Email       string
	DisplayName string
	Password    string
}

func main() {
	ctx := context.Background()

	repoMode, err := config.RepositoryMode()
	if err != nil {
		log.Fatalf("invalid repository mode: %v", err)
	}
	if repoMode != config.RepositoryPostgres {
		log.Fatalf("metaldocs-e2e-seed requires postgres repository mode")
	}

	attachmentsCfg, err := config.LoadAttachmentsConfig()
	if err != nil {
		log.Fatalf("invalid attachments config: %v", err)
	}
	authCfg, err := authn.LoadRuntimeConfig()
	if err != nil {
		log.Fatalf("invalid auth config: %v", err)
	}

	deps, err := bootstrap.BuildAPIDependencies(ctx, repoMode, attachmentsCfg)
	if err != nil {
		log.Fatalf("build api dependencies: %v", err)
	}
	defer deps.Cleanup()

	authService := authapp.NewService(deps.AuthRepo, deps.RoleProvider, deps.RoleAdminRepo, authCfg)
	iamAdmin := iamapp.NewAdminService(deps.RoleAdminRepo, nil)
	seed := loadSeedConfig()

	exists, err := userExists(ctx, authService, seed.UserID)
	if err != nil {
		log.Fatalf("check existing user: %v", err)
	}

	if !exists {
		if err := authService.CreateUser(ctx, seed.UserID, seed.Username, seed.Email, seed.DisplayName, seed.Password, []iamdomain.Role{iamdomain.RoleAdmin}, "e2e-seed"); err != nil {
			log.Fatalf("create e2e user: %v", err)
		}
	} else {
		active := true
		mustChangePassword := false
		email := seed.Email
		displayName := seed.DisplayName
		if err := authService.UpdateUser(ctx, authdomain.UpdateUserParams{
			UserID:             seed.UserID,
			DisplayName:        &displayName,
			Email:              &email,
			IsActive:           &active,
			MustChangePassword: &mustChangePassword,
		}, seed.Password); err != nil {
			log.Fatalf("reset e2e user: %v", err)
		}
	}

	if err := iamAdmin.UpsertUserAndAssignRole(ctx, seed.UserID, seed.DisplayName, iamdomain.RoleAdmin, "e2e-seed"); err != nil {
		log.Fatalf("ensure admin role: %v", err)
	}

	log.Printf("e2e seed ready user_id=%s username=%s", seed.UserID, seed.Username)
}

func loadSeedConfig() seedConfig {
	return seedConfig{
		UserID:      readEnv("METALDOCS_E2E_ADMIN_USER_ID", "e2e-admin"),
		Username:    readEnv("METALDOCS_E2E_ADMIN_USERNAME", "e2e.admin"),
		Email:       readEnv("METALDOCS_E2E_ADMIN_EMAIL", "e2e.admin@local.test"),
		DisplayName: readEnv("METALDOCS_E2E_ADMIN_DISPLAY_NAME", "E2E Admin"),
		Password:    requireEnv("METALDOCS_E2E_ADMIN_PASSWORD", "E2eAdmin123!"),
	}
}

func userExists(ctx context.Context, service *authapp.Service, userID string) (bool, error) {
	items, err := service.ListUsers(ctx)
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if strings.TrimSpace(item.UserID) == strings.TrimSpace(userID) {
			return true, nil
		}
	}
	return false, nil
}

func readEnv(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func requireEnv(name, fallback string) string {
	value := readEnv(name, fallback)
	if strings.TrimSpace(value) == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}
