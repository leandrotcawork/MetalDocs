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
	PermDocumentCreate            Permission = "document:create"
	PermDocumentRead              Permission = "document:read"
	PermDocumentManagePermissions Permission = "document:manage_permissions"
	PermVersionRead               Permission = "version:read"
	PermWorkflowTransition        Permission = "workflow:transition"
	PermSearchRead                Permission = "search:read"
	PermIAMManageRoles            Permission = "iam:manage_roles"
)
