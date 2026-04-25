package domain

import "time"

type UserProcessArea struct {
	UserID        string
	TenantID      string
	AreaCode      string
	Role          Role
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	GrantedBy     *string
}

func (m UserProcessArea) IsActive(now time.Time) bool {
	if m.EffectiveFrom.After(now) {
		return false
	}
	if m.EffectiveTo == nil {
		return true
	}
	return m.EffectiveTo.After(now)
}
