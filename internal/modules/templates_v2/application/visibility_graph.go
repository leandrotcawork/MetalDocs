package application

import (
	"fmt"
	"strings"

	"metaldocs/internal/modules/templates_v2/domain"
)

const (
	nodeWhite = 0
	nodeGray  = 1
	nodeBlack = 2
)

func DetectVisibilityCycle(phs []domain.Placeholder) error {
	placeholdersByID := make(map[string]domain.Placeholder, len(phs))
	for _, p := range phs {
		placeholdersByID[p.ID] = p
	}

	color := make(map[string]int, len(phs))
	var visit func(id string, stack []string) error
	visit = func(id string, stack []string) error {
		switch color[id] {
		case nodeGray:
			cycle := append(stack, id)
			return fmt.Errorf("visibility cycle: %s: %w", strings.Join(cycle, " -> "), domain.ErrPlaceholderCycle)
		case nodeBlack:
			return nil
		}

		color[id] = nodeGray
		stack = append(stack, id)

		p := placeholdersByID[id]
		if p.VisibleIf != nil {
			depID := p.VisibleIf.PlaceholderID
			if _, ok := placeholdersByID[depID]; !ok {
				return fmt.Errorf("placeholder[%s] unknown visibility dependency %q: %w", p.ID, depID, domain.ErrInvalidConstraint)
			}
			if err := visit(depID, stack); err != nil {
				return err
			}
		}

		color[id] = nodeBlack
		return nil
	}

	for _, p := range phs {
		if color[p.ID] == nodeWhite {
			if err := visit(p.ID, nil); err != nil {
				return err
			}
		}
	}

	return nil
}
