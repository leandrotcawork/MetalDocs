package domain

import "errors"

type TemplateResolutionInput struct {
	ProfileCode      string
	OverrideTemplate *TemplateVersionCandidate
	DefaultTemplate  *TemplateVersionCandidate
}

type TemplateVersionCandidate struct {
	ID          string
	ProfileCode string
	Status      *string
}

type TemplateResolutionResult struct {
	TemplateVersionID string
	Source            string
}

var (
	ErrProfileHasNoDefaultTemplate = errors.New("profile has no default template")
	ErrOverrideTemplateDeleted     = errors.New("override template deleted")
	ErrOverrideNotPublished        = errors.New("override template is not published")
	ErrDefaultObsolete             = errors.New("default template is obsolete")
	ErrTemplateProfileMismatch     = errors.New("template profile mismatch")
)

func Resolve(in TemplateResolutionInput) (TemplateResolutionResult, error) {
	if in.OverrideTemplate != nil {
		o := in.OverrideTemplate
		if o.Status == nil {
			return TemplateResolutionResult{}, ErrOverrideTemplateDeleted
		}
		if *o.Status != "published" {
			return TemplateResolutionResult{}, ErrOverrideNotPublished
		}
		if o.ProfileCode != in.ProfileCode {
			return TemplateResolutionResult{}, ErrTemplateProfileMismatch
		}
		return TemplateResolutionResult{TemplateVersionID: o.ID, Source: "override"}, nil
	}

	if in.DefaultTemplate == nil {
		return TemplateResolutionResult{}, ErrProfileHasNoDefaultTemplate
	}
	d := in.DefaultTemplate
	if d.Status == nil {
		return TemplateResolutionResult{}, ErrProfileHasNoDefaultTemplate
	}
	if *d.Status == "obsolete" {
		return TemplateResolutionResult{}, ErrDefaultObsolete
	}
	if *d.Status != "published" {
		return TemplateResolutionResult{}, ErrProfileHasNoDefaultTemplate
	}
	return TemplateResolutionResult{TemplateVersionID: d.ID, Source: "default"}, nil
}
