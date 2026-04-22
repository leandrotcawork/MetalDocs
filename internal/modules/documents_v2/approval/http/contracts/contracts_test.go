package contracts

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSubmitRequestValidate(t *testing.T) {
	valid := SubmitRequest{RouteID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", ContentHash: strings.Repeat("a", 64)}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	missingRoute := SubmitRequest{ContentHash: strings.Repeat("a", 64)}
	if err := missingRoute.Validate(); err == nil {
		t.Fatalf("expected error for missing route_id")
	}

	badHash := SubmitRequest{RouteID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", ContentHash: "abc"}
	if err := badHash.Validate(); err == nil {
		t.Fatalf("expected error for invalid content_hash")
	}
}

func TestSignoffRequestValidate(t *testing.T) {
	valid := SignoffRequest{
		Decision:      "approve",
		PasswordToken: "token",
		ContentHash:   strings.Repeat("b", 64),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	invalidDecision := SignoffRequest{Decision: "maybe", PasswordToken: "token", ContentHash: strings.Repeat("b", 64)}
	if err := invalidDecision.Validate(); err == nil {
		t.Fatalf("expected error for invalid decision")
	}

	missingRejectReason := SignoffRequest{Decision: "reject", PasswordToken: "token", ContentHash: strings.Repeat("b", 64)}
	if err := missingRejectReason.Validate(); err == nil {
		t.Fatalf("expected error for missing reason on reject")
	}

	missingPassword := SignoffRequest{Decision: "approve", ContentHash: strings.Repeat("b", 64)}
	if err := missingPassword.Validate(); err == nil {
		t.Fatalf("expected error for missing password_token")
	}
}

func TestSchedulePublishRequestValidate(t *testing.T) {
	valid := SchedulePublishRequest{EffectiveFrom: "2026-12-31T18:00:00Z"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	missing := SchedulePublishRequest{}
	if err := missing.Validate(); err == nil {
		t.Fatalf("expected error for missing effective_from")
	}

	invalid := SchedulePublishRequest{EffectiveFrom: "31-12-2026"}
	if err := invalid.Validate(); err == nil {
		t.Fatalf("expected error for invalid effective_from")
	}

	nonUTC := SchedulePublishRequest{EffectiveFrom: "2026-12-31T18:00:00-03:00"}
	if err := nonUTC.Validate(); err == nil {
		t.Fatalf("expected error for non-UTC effective_from")
	}
}

func TestSupersedeRequestValidate(t *testing.T) {
	valid := SupersedeRequest{SupersededDocumentID: "3fa85f64-5717-4562-b3fc-2c963f66afa6"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	missing := SupersedeRequest{}
	if err := missing.Validate(); err == nil {
		t.Fatalf("expected error for missing superseded_document_id")
	}
}

func TestObsoleteRequestValidate(t *testing.T) {
	valid := ObsoleteRequest{Reason: "Replaced by newer standard"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	missing := ObsoleteRequest{}
	if err := missing.Validate(); err == nil {
		t.Fatalf("expected error for missing reason")
	}
}

func TestCancelRequestValidate(t *testing.T) {
	valid := CancelRequest{Reason: "Withdrawn"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	missing := CancelRequest{}
	if err := missing.Validate(); err == nil {
		t.Fatalf("expected error for missing reason")
	}
}

func TestCreateRouteRequestValidate(t *testing.T) {
	m := 2
	valid := CreateRouteRequest{
		ProfileCode: "ops",
		Name:        "Default Route",
		Stages: []StageRequest{
			{
				Order:              1,
				Name:               "Review",
				RequiredRole:       "reviewer",
				RequiredCapability: "doc.signoff",
				AreaCode:           "ops",
				Quorum:             "any_1_of",
				DriftPolicy:        "reduce_quorum",
			},
			{
				Order:              2,
				Name:               "Approval",
				RequiredRole:       "approver",
				RequiredCapability: "doc.signoff",
				AreaCode:           "ops",
				Quorum:             "m_of_n",
				QuorumM:            &m,
				DriftPolicy:        "keep_snapshot",
			},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	missingStages := CreateRouteRequest{ProfileCode: "ops", Name: "x"}
	if err := missingStages.Validate(); err == nil {
		t.Fatalf("expected error for empty stages")
	}

	badQuorum := valid
	badQuorum.Stages[0].Quorum = "half"
	if err := badQuorum.Validate(); err == nil {
		t.Fatalf("expected error for invalid quorum")
	}

	badOrder := valid
	badOrder.Stages[0].Order = 2
	if err := badOrder.Validate(); err == nil {
		t.Fatalf("expected error for invalid order")
	}
}

func TestUpdateRouteRequestValidate(t *testing.T) {
	valid := UpdateRouteRequest{
		Name: "Updated Route",
		Stages: []StageRequest{
			{
				Order:              1,
				Name:               "Review",
				RequiredRole:       "reviewer",
				RequiredCapability: "doc.signoff",
				AreaCode:           "ops",
				Quorum:             "all_of",
				DriftPolicy:        "fail_stage",
			},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	missingName := valid
	missingName.Name = ""
	if err := missingName.Validate(); err == nil {
		t.Fatalf("expected error for missing name")
	}

	badDrift := valid
	badDrift.Stages[0].DriftPolicy = "ignore"
	if err := badDrift.Validate(); err == nil {
		t.Fatalf("expected error for invalid drift_policy")
	}
}

func TestDecode(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		var dst struct {
			RouteID string `json:"route_id"`
		}
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"route_id":"abc"}`))
		req.Header.Set("Content-Type", "application/json")
		if err := Decode(req, &dst); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("wrong content type", func(t *testing.T) {
		var dst map[string]any
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"a":1}`))
		req.Header.Set("Content-Type", "text/plain")
		if err := Decode(req, &dst); !errors.Is(err, ErrContentType) {
			t.Fatalf("expected ErrContentType, got %v", err)
		}
	})

	t.Run("body too large", func(t *testing.T) {
		var dst map[string]any
		payload := `{"v":"` + strings.Repeat("a", 70*1024) + `"}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		if err := Decode(req, &dst); !errors.Is(err, ErrBodyTooLarge) {
			t.Fatalf("expected ErrBodyTooLarge, got %v", err)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		var dst map[string]any
		req := httptest.NewRequest("POST", "/", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/json")
		if err := Decode(req, &dst); !errors.Is(err, ErrEmptyBody) {
			t.Fatalf("expected ErrEmptyBody, got %v", err)
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		var dst map[string]any
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"a":}`))
		req.Header.Set("Content-Type", "application/json")
		err := Decode(req, &dst)
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Fatalf("expected *json.SyntaxError, got %T (%v)", err, err)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		var dst struct {
			RouteID string `json:"route_id"`
		}
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"route_id":"abc","unknown":1}`))
		req.Header.Set("Content-Type", "application/json")
		err := Decode(req, &dst)
		if err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("expected unknown field error, got %v", err)
		}
	})
}
