import type { DocumentDepartmentItem, ProcessAreaItem } from "../../lib.types";
import type { DocumentForm } from "./documentCreateTypes";
import { CreateField } from "./widgets/CreateField";
import { DateTimeField } from "./widgets/DateTimeField";
import { FilterDropdown, type SelectMenuOption } from "../ui/FilterDropdown";

type DocumentCreateContentStepProps = {
  form: DocumentForm;
  processAreas: ProcessAreaItem[];
  documentDepartments: DocumentDepartmentItem[];
  onDocumentFormChange: (next: DocumentForm) => void;
};

const classificationOptions = [
  { value: "PUBLIC", label: "Publico", desc: "Visivel externamente" },
  { value: "INTERNAL", label: "Interno", desc: "Todos os usuarios da empresa" },
  { value: "CONFIDENTIAL", label: "Departamentos", desc: "Departamentos selecionados" },
  { value: "RESTRICTED", label: "Restrito", desc: "Departamento + area especifica" },
];

export function DocumentCreateContentStep(props: DocumentCreateContentStepProps) {
  const isConfidential = props.form.classification === "CONFIDENTIAL";
  const isRestricted = props.form.classification === "RESTRICTED";
  const showAudience = isConfidential || isRestricted;
  const departmentOptions: SelectMenuOption[] = [
    ...props.documentDepartments.map((item) => ({
      value: item.code,
      label: item.name,
    })),
  ];
  const confidentialDepartmentOptions: SelectMenuOption[] = [
    ...departmentOptions,
  ];
  const processAreaOptions: SelectMenuOption[] = [
    ...props.processAreas.map((item) => ({
      value: item.code,
      label: item.name,
    })),
  ];

  return (
    <div className="stack">
      <div className="create-doc-subsection">
        <h4 className="create-doc-subsection-title">Classificacao e acesso</h4>
        <div className="field">
          <label className="field-label"><span>Tipo de acesso</span></label>
          <div className="class-chips">
            {classificationOptions.map((item) => (
              <button
                key={item.value}
                type="button"
                className={`class-chip ${props.form.classification === item.value ? "active" : ""}`}
                data-classification={item.value}
                onClick={() => {
                  const nextForm = { ...props.form, classification: item.value };
                  if (item.value === "CONFIDENTIAL" || item.value === "RESTRICTED") {
                    if (!nextForm.audienceMode || nextForm.audienceMode === "INTERNAL" || item.value === "RESTRICTED") {
                      nextForm.audienceMode = item.value === "RESTRICTED" ? "AREAS" : "DEPARTMENT";
                    }
                    if (!nextForm.audienceDepartment) {
                      nextForm.audienceDepartment = nextForm.department;
                    }
                    if (item.value === "CONFIDENTIAL" && nextForm.audienceDepartments.length === 0 && nextForm.department) {
                      nextForm.audienceDepartments = [nextForm.department];
                    }
                    if (item.value === "RESTRICTED" && !nextForm.audienceProcessArea) {
                      nextForm.audienceProcessArea = nextForm.processArea;
                    }
                  } else {
                    nextForm.audienceMode = "INTERNAL";
                  }
                  props.onDocumentFormChange(nextForm);
                }}
              >
                <span className="class-chip-title-row">
                  <span className="class-chip-dot" aria-hidden />
                  <span className="class-chip-title">{item.label}</span>
                </span>
                <small className="class-chip-desc">{item.desc}</small>
              </button>
            ))}
          </div>
        </div>

        {showAudience && (
          <div className="catalog-form-grid">
            {isConfidential ? (
              <CreateField label="Departamentos">
                <FilterDropdown
                  id="document-audience-departments"
                  value=""
                  values={props.form.audienceDepartments}
                  placeholder="Selecione departamentos"
                  options={confidentialDepartmentOptions}
                  selectionMode="duo"
                  searchThreshold={0}
                  closeOnSelectInDuo={false}
                  onSelect={(value) => {
                    if (!value) return;
                    const current = props.form.audienceDepartments;
                    const next = current.includes(value)
                      ? current.filter((item) => item !== value)
                      : [...current, value];
                    props.onDocumentFormChange({ ...props.form, audienceDepartments: next });
                  }}
                />
              </CreateField>
            ) : (
              <>
                <CreateField label="Departamento">
                  <FilterDropdown
                    id="document-audience-department"
                    value={props.form.audienceDepartment}
                    placeholder="Selecione o departamento"
                    options={departmentOptions}
                    onSelect={(value) => props.onDocumentFormChange({ ...props.form, audienceDepartment: value })}
                  />
                </CreateField>
                {props.form.audienceMode === "AREAS" && (
                  <CreateField label="Area de processo">
                    <FilterDropdown
                      id="document-audience-process-area"
                      value={props.form.audienceProcessArea}
                      placeholder="Selecione a area"
                      options={processAreaOptions}
                      onSelect={(value) => props.onDocumentFormChange({ ...props.form, audienceProcessArea: value })}
                    />
                  </CreateField>
                )}
              </>
            )}
          </div>
        )}
        <div className="catalog-form-grid">
          <CreateField label="Inicio de vigencia">
            <DateTimeField
              id="document-effective-at"
              value={props.form.effectiveAt}
              onChange={(value) => props.onDocumentFormChange({ ...props.form, effectiveAt: value })}
            />
          </CreateField>
          <CreateField label="Fim de vigencia">
            <DateTimeField
              id="document-expiry-at"
              value={props.form.expiryAt}
              onChange={(value) => props.onDocumentFormChange({ ...props.form, expiryAt: value })}
            />
          </CreateField>
        </div>
      </div>
    </div>
  );
}
