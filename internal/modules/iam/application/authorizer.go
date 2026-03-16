package application

import "metaldocs/internal/modules/iam/domain"

type StaticAuthorizer struct {
	policy map[domain.Role]map[domain.Permission]bool
}

func NewStaticAuthorizer() *StaticAuthorizer {
	return &StaticAuthorizer{
		policy: map[domain.Role]map[domain.Permission]bool{
			domain.RoleAdmin: {
				domain.PermDocumentCreate: true,
				domain.PermDocumentRead:   true,
				domain.PermVersionRead:    true,
			},
			domain.RoleEditor: {
				domain.PermDocumentCreate: true,
				domain.PermDocumentRead:   true,
				domain.PermVersionRead:    true,
			},
			domain.RoleReviewer: {
				domain.PermDocumentRead: true,
				domain.PermVersionRead:  true,
			},
			domain.RoleViewer: {
				domain.PermDocumentRead: true,
				domain.PermVersionRead:  true,
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
