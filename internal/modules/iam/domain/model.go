package domain

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleEditor   Role = "editor"
	RoleReviewer Role = "reviewer"
	RoleViewer   Role = "viewer"
)

type Permission string

const (
	PermDocumentCreate Permission = "document:create"
	PermDocumentRead   Permission = "document:read"
	PermVersionRead    Permission = "version:read"
	PermIAMManageRoles Permission = "iam:manage_roles"
)
