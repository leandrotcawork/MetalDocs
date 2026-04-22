package contracts

import "fmt"

type SignoffRequest struct {
	Decision       string `json:"decision"`
	Reason         string `json:"reason"`
	PasswordToken  string `json:"password_token"`
	ContentHash    string `json:"content_hash"`
	IdempotencyKey string
}

func (r SignoffRequest) Validate() error {
	if r.Decision != "approve" && r.Decision != "reject" {
		return fmt.Errorf("decision must be one of: approve, reject")
	}
	if r.Decision == "reject" {
		if err := validateRequired("reason", r.Reason); err != nil {
			return err
		}
	}
	if err := validateRequired("password_token", r.PasswordToken); err != nil {
		return err
	}
	if err := validateSHA256Hex("content_hash", r.ContentHash); err != nil {
		return err
	}
	return nil
}

type SignoffResponse struct {
	SignoffID string `json:"signoff_id"`
	WasReplay bool   `json:"was_replay"`
	Outcome   string `json:"outcome"`
}
