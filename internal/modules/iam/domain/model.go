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
	PermDocumentEdit              Permission = "document:edit"
	PermDocumentRead              Permission = "document:read"
	PermDocumentUploadAttachment  Permission = "document:upload_attachment"
	PermDocumentManagePermissions Permission = "document:manage_permissions"
	PermVersionRead               Permission = "version:read"
	PermWorkflowTransition        Permission = "workflow:transition"
	PermSearchRead                Permission = "search:read"
	PermIAMManageRoles            Permission = "iam:manage_roles"
	PermTemplateView              Permission = "template:view"
	PermTemplateEdit              Permission = "template:edit"
	PermTemplatePublish           Permission = "template:publish"
	PermTemplateExport            Permission = "template:export"
)
