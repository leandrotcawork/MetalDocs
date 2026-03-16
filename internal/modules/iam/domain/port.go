package domain

import "context"

// Authorizer is the stable IAM contract for authorization decisions.
type Authorizer interface {
	Can(role Role, permission Permission) bool
}

// RoleProvider resolves effective roles for a given user identity.
type RoleProvider interface {
	RolesByUserID(ctx context.Context, userID string) ([]Role, error)
}

// RoleAdminRepository writes IAM user and role assignments.
type RoleAdminRepository interface {
	UpsertUserAndAssignRole(ctx context.Context, userID, displayName string, role Role, assignedBy string) error
}
