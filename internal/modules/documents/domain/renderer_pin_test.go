package domain

import "testing"

func TestRendererPin_IsComplete(t *testing.T) {
	pin := RendererPin{
		RendererVersion: "1.0.0",
		LayoutIRHash:    "abc123",
		TemplateKey:     "po-mddm-canvas",
		TemplateVersion: 1,
	}
	if !pin.IsComplete() {
		t.Fatalf("expected pin to be complete")
	}

	incomplete := RendererPin{RendererVersion: "1.0.0"}
	if incomplete.IsComplete() {
		t.Fatalf("expected pin missing fields to be incomplete")
	}
}

func TestRendererPin_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pin     RendererPin
		wantErr bool
	}{
		{
			name:    "valid pin",
			pin:     RendererPin{RendererVersion: "1.0.0", LayoutIRHash: "abcdef", TemplateKey: "po", TemplateVersion: 1},
			wantErr: false,
		},
		{
			name:    "missing renderer version",
			pin:     RendererPin{LayoutIRHash: "abc", TemplateKey: "po", TemplateVersion: 1},
			wantErr: true,
		},
		{
			name:    "zero template version",
			pin:     RendererPin{RendererVersion: "1.0.0", LayoutIRHash: "abc", TemplateKey: "po", TemplateVersion: 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pin.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
