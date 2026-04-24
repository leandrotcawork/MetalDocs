import { useState } from 'react';
import type { Placeholder } from '../templates/placeholder-types';
import { submitDocument } from './v2/api/documentsV2';
import type { PlaceholderValueDTO } from './v2/api/documentsV2';

interface SubmitButtonProps {
  docId: string;
  placeholderSchema: Placeholder[];
  placeholderValues: PlaceholderValueDTO[];
  onSubmitted?: () => void;
}

export function SubmitButton({
  docId,
  placeholderSchema,
  placeholderValues,
  onSubmitted,
}: SubmitButtonProps) {
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const missingRequired = placeholderSchema.filter(
    (p) =>
      p.required &&
      !placeholderValues.find((v) => v.placeholder_id === p.id && v.value_text),
  );

  const disabled = missingRequired.length > 0 || submitting;

  async function handleClick() {
    if (disabled) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      await submitDocument(docId);
      onSubmitted?.();
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <button
        type="button"
        data-testid="submit-btn"
        disabled={disabled}
        onClick={() => void handleClick()}
      >
        {submitting ? 'Submitting…' : 'Submit for Approval'}
      </button>
      {missingRequired.length > 0 && (
        <ul data-testid="missing-required-list">
          {missingRequired.map((p) => (
            <li key={p.id}>{p.label}</li>
          ))}
        </ul>
      )}
      {submitError && <div role="alert">{submitError}</div>}
    </div>
  );
}
