package application

// planReleaseSteps returns the ordered list of operations needed to release a draft.
// The order matters because the partial unique index allows only one 'released' status.
func planReleaseSteps(documentID, draftID, prevReleasedID string) []string {
	steps := []string{}
	if prevReleasedID != "" {
		steps = append(steps, "archive_previous_released")
	}
	steps = append(steps, "promote_draft_to_released")
	steps = append(steps, "compute_and_store_diff")
	steps = append(steps, "delete_archived_image_refs")
	steps = append(steps, "cascade_orphan_image_cleanup")
	return steps
}
