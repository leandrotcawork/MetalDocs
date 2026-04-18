package domain

// CanTransitionDocument returns true iff a document can move from cur to next.
// draft -> finalized | archived
// finalized -> archived
// archived -> (terminal)
func CanTransitionDocument(cur, next DocumentStatus) bool {
	switch cur {
	case DocStatusDraft:
		return next == DocStatusFinalized || next == DocStatusArchived
	case DocStatusFinalized:
		return next == DocStatusArchived
	default:
		return false
	}
}
