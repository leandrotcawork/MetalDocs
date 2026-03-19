import type { DocumentForm } from "./documentCreateTypes";
import { CreateField } from "./widgets/CreateField";

type DocumentCreateContentStepProps = {
  form: DocumentForm;
  onDocumentFormChange: (next: DocumentForm) => void;
};

const classificationOptions = [
  { value: "PUBLIC", label: "Publico", desc: "Visivel externamente" },
  { value: "INTERNAL", label: "Interno", desc: "Apenas colaboradores" },
  { value: "CONFIDENTIAL", label: "Confidencial", desc: "Acesso restrito" },
  { value: "RESTRICTED", label: "Restrito", desc: "Somente autorizados" },
];

export function DocumentCreateContentStep(props: DocumentCreateContentStepProps) {
  return (
    <div className="stack">
      <div className="field">
        <label className="field-label"><span>Classificacao</span></label>
        <div className="class-chips">
          {classificationOptions.map((item) => (
            <button
              key={item.value}
              type="button"
              className={`class-chip ${props.form.classification === item.value ? "active" : ""}`}
              onClick={() => props.onDocumentFormChange({ ...props.form, classification: item.value })}
            >
              <span>{item.label}</span>
              <small>{item.desc}</small>
            </button>
          ))}
        </div>
      </div>

      <div className="catalog-form-grid">
        <CreateField label="Inicio de vigencia">
          <input
            id="document-effective-at"
            type="datetime-local"
            value={props.form.effectiveAt}
            onChange={(event) => props.onDocumentFormChange({ ...props.form, effectiveAt: event.target.value })}
          />
        </CreateField>
        <CreateField label="Fim de vigencia">
          <input
            id="document-expiry-at"
            type="datetime-local"
            value={props.form.expiryAt}
            onChange={(event) => props.onDocumentFormChange({ ...props.form, expiryAt: event.target.value })}
          />
        </CreateField>
      </div>

      <div className="catalog-form-grid">
        <CreateField label="Tags" hint="Separar por virgula">
          <input
            id="document-tags"
            data-testid="document-tags"
            placeholder="Tags separadas por virgula"
            value={props.form.tags}
            onChange={(event) => props.onDocumentFormChange({ ...props.form, tags: event.target.value })}
          />
        </CreateField>
      </div>
      <CreateField label="Conteudo inicial">
        <textarea
          data-testid="document-initial-content"
          rows={14}
          value={props.form.initialContent}
          onChange={(event) => props.onDocumentFormChange({ ...props.form, initialContent: event.target.value })}
          placeholder="Conteudo inicial da versao 1"
        />
      </CreateField>
    </div>
  );
}
