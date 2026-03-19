import type { DocumentDepartmentItem, ProcessAreaItem } from "../../lib.types";
import type { DocumentForm } from "./documentCreateTypes";
import { CreateField } from "./widgets/CreateField";
import { FilterDropdown, type SelectMenuOption } from "../ui/FilterDropdown";

type DocumentCreateContentStepProps = {
  form: DocumentForm;
  processAreas: ProcessAreaItem[];
  documentDepartments: DocumentDepartmentItem[];
  onDocumentFormChange: (next: DocumentForm) => void;
};

const classificationOptions = [
  { value: "PUBLIC", label: "Publico", desc: "Visivel externamente quando publicado" },
  { value: "INTERNAL", label: "Interno", desc: "Todos os usuarios da empresa" },
  { value: "CONFIDENTIAL", label: "Confidencial", desc: "Acesso controlado por equipes" },
  { value: "RESTRICTED", label: "Restrito", desc: "Acesso minimo e validado" },
];

export function DocumentCreateContentStep(props: DocumentCreateContentStepProps) {
  const showAudience = props.form.classification === "CONFIDENTIAL" || props.form.classification === "RESTRICTED";
  const audienceModeOptions: SelectMenuOption[] = [
    { value: "DEPARTMENT", label: "Departamento" },
    { value: "AREAS", label: "Areas do departamento" },
  ];
  const departmentOptions: SelectMenuOption[] = [
    { value: "", label: "Selecione o departamento" },
    ...props.documentDepartments.map((item) => ({
      value: item.code,
      label: item.name,
    })),
  ];
  const processAreaOptions: SelectMenuOption[] = [
    { value: "", label: "Selecione a area" },
    ...props.processAreas.map((item) => ({
      value: item.code,
      label: item.name,
    })),
  ];

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
              onClick={() => {
                const nextForm = { ...props.form, classification: item.value };
                if (item.value === "CONFIDENTIAL" || item.value === "RESTRICTED") {
                  if (!nextForm.audienceMode || nextForm.audienceMode === "INTERNAL") {
                    nextForm.audienceMode = "DEPARTMENT";
                  }
                  if (!nextForm.audienceDepartment) {
                    nextForm.audienceDepartment = nextForm.department;
                  }
                } else {
                  nextForm.audienceMode = "INTERNAL";
                }
                props.onDocumentFormChange(nextForm);
              }}
            >
              <span>{item.label}</span>
              <small>{item.desc}</small>
            </button>
          ))}
        </div>
      </div>

      {showAudience && (
        <div className="catalog-form-grid">
          <CreateField label="Quem pode ver">
            <FilterDropdown
              id="document-audience-mode"
              value={props.form.audienceMode}
              options={audienceModeOptions}
              onSelect={(value) => {
                const nextMode = value || "DEPARTMENT";
                const nextForm = { ...props.form, audienceMode: nextMode };
                if (!nextForm.audienceDepartment) {
                  nextForm.audienceDepartment = nextForm.department;
                }
                if (nextMode === "AREAS" && !nextForm.audienceProcessArea) {
                  nextForm.audienceProcessArea = nextForm.processArea;
                }
                props.onDocumentFormChange(nextForm);
              }}
            />
          </CreateField>
          <CreateField label="Departamento">
            <FilterDropdown
              id="document-audience-department"
              value={props.form.audienceDepartment}
              options={departmentOptions}
              onSelect={(value) => props.onDocumentFormChange({ ...props.form, audienceDepartment: value })}
            />
          </CreateField>
          {props.form.audienceMode === "AREAS" && (
            <CreateField label="Area de processo">
              <FilterDropdown
                id="document-audience-process-area"
                value={props.form.audienceProcessArea}
                options={processAreaOptions}
                onSelect={(value) => props.onDocumentFormChange({ ...props.form, audienceProcessArea: value })}
              />
            </CreateField>
          )}
        </div>
      )}

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
