import { metalNobreProcessAreaHint, metalNobreProcessAreaOptionLabel } from "../../features/documents/adapters/metalNobreExperience";
import type { DocumentDepartmentItem, ProcessAreaItem, SubjectItem } from "../../lib.types";
import type { DocumentForm } from "./documentCreateTypes";
import { CreateField } from "./widgets/CreateField";
import { FilterDropdown, type SelectMenuOption } from "../ui/FilterDropdown";

type DocumentCreateContextStepProps = {
  form: DocumentForm;
  processAreas: ProcessAreaItem[];
  documentDepartments: DocumentDepartmentItem[];
  subjects: SubjectItem[];
  onDocumentFormChange: (next: DocumentForm) => void;
};

export function DocumentCreateContextStep(props: DocumentCreateContextStepProps) {
  const availableSubjects = props.subjects.filter((item) => !props.form.processArea || item.processAreaCode === props.form.processArea);
  const processAreaOptions: SelectMenuOption[] = [
    { value: "", label: "Sem process area" },
    ...props.processAreas.map((item) => ({
      value: item.code,
      label: metalNobreProcessAreaOptionLabel(item),
    })),
  ];
  const subjectOptions: SelectMenuOption[] = [
    { value: "", label: "Sem subject" },
    ...availableSubjects.map((item) => ({
      value: item.code,
      label: item.name,
    })),
  ];
  const departmentOptions: SelectMenuOption[] = [
    { value: "", label: "Selecione o departamento" },
    ...props.documentDepartments.map((item) => ({
      value: item.code,
      label: item.name,
    })),
  ];

  return (
    <div className="stack">
      <div className="catalog-form-grid">
        <CreateField label="Responsavel (Owner)" required hint="Preenchido automaticamente pelo usuario logado.">
          <input
            id="document-owner"
            data-testid="document-owner"
            placeholder="Owner"
            value={props.form.ownerId}
            onChange={(event) => props.onDocumentFormChange({ ...props.form, ownerId: event.target.value })}
            readOnly
            disabled
          />
        </CreateField>
        <CreateField label="Departamento" required>
          <FilterDropdown
            id="document-department"
            value={props.form.department}
            options={departmentOptions}
            onSelect={(value) => props.onDocumentFormChange({ ...props.form, department: value })}
          />
        </CreateField>
        <CreateField
          label="Area de processo"
          hint={props.form.processArea ? metalNobreProcessAreaHint(props.form.processArea) : "Selecione a area para guiar subjects e classificacao operacional."}
        >
          <FilterDropdown
            id="document-process-area"
            value={props.form.processArea}
            options={processAreaOptions}
            onSelect={(value) => {
              const areaLabel = value
                ? (props.processAreas.find((item) => item.code === value)?.name ?? value)
                : "Metal Nobre";
              props.onDocumentFormChange({
                ...props.form,
                processArea: value,
                subject: "",
                businessUnit: areaLabel,
              });
            }}
          />
        </CreateField>
        <CreateField label="Assunto (Subject)">
          <FilterDropdown
            id="document-subject"
            value={props.form.subject}
            options={subjectOptions}
            onSelect={(value) => props.onDocumentFormChange({ ...props.form, subject: value })}
          />
        </CreateField>
      </div>
    </div>
  );
}
