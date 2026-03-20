package carbone

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"metaldocs/internal/platform/config"
)

type TemplateBinding struct {
	ProfileCode string
	FileName    string
}

type TemplateRegistry struct {
	templates map[string]string
	root      string
}

func DefaultTemplateBindings() []TemplateBinding {
	return []TemplateBinding{
		{ProfileCode: "po", FileName: "template-po.docx"},
		{ProfileCode: "it", FileName: "template-it.docx"},
		{ProfileCode: "rg", FileName: "template-rg.docx"},
		{ProfileCode: "fm", FileName: "template-fm.docx"},
	}
}

func BootstrapTemplates(ctx context.Context, client *Client, cfg config.CarboneConfig, bindings []TemplateBinding) (*TemplateRegistry, error) {
	registry := &TemplateRegistry{
		templates: make(map[string]string),
		root:      cfg.TemplateRoot,
	}
	if !cfg.Enabled {
		return registry, nil
	}
	if client == nil {
		return registry, fmt.Errorf("carbone client not configured")
	}
	if len(bindings) == 0 {
		bindings = DefaultTemplateBindings()
	}

	var errs []error
	for _, binding := range bindings {
		profileCode := strings.ToLower(strings.TrimSpace(binding.ProfileCode))
		if profileCode == "" {
			continue
		}
		fileName := strings.TrimSpace(binding.FileName)
		if fileName == "" {
			errs = append(errs, fmt.Errorf("template binding missing file for profile %s", profileCode))
			continue
		}
		filePath := filepath.Join(cfg.TemplateRoot, fileName)
		if _, err := os.Stat(filePath); err != nil {
			errs = append(errs, fmt.Errorf("template missing profile=%s file=%s: %w", profileCode, filePath, err))
			continue
		}
		templateID, err := client.RegisterTemplate(ctx, "bootstrap", filePath)
		if err != nil {
			errs = append(errs, fmt.Errorf("register template profile=%s: %w", profileCode, err))
			continue
		}
		registry.templates[profileCode] = templateID
		log.Printf("carbone template registered profile=%s template_id=%s", profileCode, templateID)
	}

	if len(errs) > 0 {
		return registry, errors.Join(errs...)
	}
	return registry, nil
}

func (r *TemplateRegistry) TemplateID(profileCode string) (string, bool) {
	if r == nil {
		return "", false
	}
	key := strings.ToLower(strings.TrimSpace(profileCode))
	value, ok := r.templates[key]
	return value, ok
}

func (r *TemplateRegistry) Count() int {
	if r == nil {
		return 0
	}
	return len(r.templates)
}

func (r *TemplateRegistry) Snapshot() map[string]string {
	if r == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(r.templates))
	for key, value := range r.templates {
		out[key] = value
	}
	return out
}
