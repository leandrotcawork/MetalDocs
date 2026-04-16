export type TemplateDraftStatus = 'draft' | 'pending_review' | 'published';

export interface TemplateDraftStatusResponse {
  draft_status: TemplateDraftStatus;
}

/**
 * POST /api/v1/templates/{key}/submit-review
 * Transitions draft → pending_review.
 * Throws on non-2xx.
 */
export async function submitTemplateForReview(templateKey: string): Promise<void> {
  const res = await fetch(`/api/v1/templates/${encodeURIComponent(templateKey)}/submit-review`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!res.ok) {
    throw new Error(`submit-review failed: ${res.status}`);
  }
}

/**
 * POST /api/v1/templates/{key}/approve
 * Transitions pending_review → published.
 * Throws on non-2xx.
 */
export async function approveTemplate(templateKey: string): Promise<void> {
  const res = await fetch(`/api/v1/templates/${encodeURIComponent(templateKey)}/approve`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!res.ok) {
    throw new Error(`approve failed: ${res.status}`);
  }
}
