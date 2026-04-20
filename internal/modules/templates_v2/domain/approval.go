package domain

type ApprovalConfig struct {
	TemplateID   string
	ReviewerRole *string
	ApproverRole string
}

func (c ApprovalConfig) HasReviewer() bool { return c.ReviewerRole != nil && *c.ReviewerRole != "" }

// CheckSegregation enforces ISO segregation of duties.
// role = "reviewer" | "approver"
// Rules:
//   role="reviewer": actorID != authorID
//   role="approver": actorID != authorID AND (reviewerID == nil OR actorID != *reviewerID)
// Returns ErrISOSegregationViolation on conflict.
func CheckSegregation(role string, actorID, authorID string, reviewerID *string) error {
	switch role {
	case "reviewer":
		if actorID == authorID {
			return ErrISOSegregationViolation
		}
		return nil
	case "approver":
		if actorID == authorID {
			return ErrISOSegregationViolation
		}
		if reviewerID != nil && actorID == *reviewerID {
			return ErrISOSegregationViolation
		}
		return nil
	default:
		return ErrForbiddenRole
	}
}
