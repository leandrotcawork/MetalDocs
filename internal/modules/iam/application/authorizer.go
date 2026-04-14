package application

import "metaldocs/internal/modules/iam/domain"

type StaticAuthorizer struct {
	policy map[domain.Role]map[domain.Permission]bool
}

func NewStaticAuthorizer() *StaticAuthorizer {
	return &StaticAuthorizer{
		policy: map[domain.Role]map[domain.Permission]bool{
			domain.RoleAdmin: {
				domain.PermDocumentCreate:            true,
				domain.PermDocumentEdit:              true,
				domain.PermDocumentRead:              true,
				domain.PermDocumentUploadAttachment:  true,
				domain.PermDocumentManagePermissions: true,
				domain.PermVersionRead:               true,
				domain.PermWorkflowTransition:        true,
				domain.PermSearchRead:                true,
				domain.PermIAMManageRoles:            true,
				domain.PermTemplateView:              true,
				domain.PermTemplateEdit:              true,
				domain.PermTemplatePublish:           true,
				domain.PermTemplateExport:            true,
			},
			domain.RoleEditor: {
				domain.PermDocumentCreate:           true,
				domain.PermDocumentEdit:             true,
				domain.PermDocumentRead:             true,
				domain.PermDocumentUploadAttachment: true,
				domain.PermVersionRead:              true,
				domain.PermWorkflowTransition:       true,
				domain.PermSearchRead:               true,
				domain.PermTemplateView:             true,
				domain.PermTemplateExport:           true,
			},
			domain.RoleReviewer: {
				domain.PermDocumentRead:       true,
				domain.PermVersionRead:        true,
				domain.PermWorkflowTransition: true,
				domain.PermSearchRead:         true,
				domain.PermTemplateView:       true,
				domain.PermTemplateExport:     true,
			},
			domain.RoleViewer: {
				domain.PermDocumentRead: true,
				domain.PermVersionRead:  true,
				domain.PermSearchRead:   true,
			},
		},
	}
}

func (a *StaticAuthorizer) Can(role domain.Role, permission domain.Permission) bool {
	rolePolicy, ok := a.policy[role]
	if !ok {
		return false
	}
	return rolePolicy[permission]
}
