package domain

import "errors"

var (
	ErrAuthorCannotSign  = errors.New("SoD: document author cannot sign their own revision")
	ErrActorAlreadySigned = errors.New("SoD: actor has already signed in a prior stage of this instance")
)

// CheckSoD validates Separation-of-Duties rules. Pure function — no DB, no globals.
//
// Rules:
//   - actorUserID must not equal authorUserID
//   - actorUserID must not appear in any priorSignoffs (same instance, earlier stages)
func CheckSoD(authorUserID string, actorUserID string, priorSignoffs []Signoff) error {
	if actorUserID == authorUserID {
		return ErrAuthorCannotSign
	}
	for _, s := range priorSignoffs {
		if s.ActorUserID() == actorUserID {
			return ErrActorAlreadySigned
		}
	}
	return nil
}
