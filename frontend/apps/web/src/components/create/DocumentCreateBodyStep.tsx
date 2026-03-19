import type { DocumentForm } from "./documentCreateTypes";
import { CreateField } from "./widgets/CreateField";

type DocumentCreateBodyStepProps = {
  form: DocumentForm;
  onDocumentFormChange: (next: DocumentForm) => void;
};

export function DocumentCreateBodyStep(props: DocumentCreateBodyStepProps) {
  return (
    <CreateField label="Conteudo inicial">
      <textarea
        data-testid="document-initial-content"
        rows={14}
        value={props.form.initialContent}
        onChange={(event) => props.onDocumentFormChange({ ...props.form, initialContent: event.target.value })}
        placeholder="Conteudo inicial da versao 1"
      />
    </CreateField>
  );
}
