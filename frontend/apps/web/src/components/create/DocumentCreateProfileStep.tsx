import {
  metalNobreProfileContext,
  metalNobreProfileOptionLabel,
} from "../../features/documents/adapters/metalNobreExperience";
import type { DocumentProfileItem } from "../../lib.types";
import type { DocumentForm } from "./documentCreateTypes";
import { CreateField } from "./widgets/CreateField";
import { FilterDropdown, type SelectMenuOption } from "../ui/FilterDropdown";

type DocumentCreateProfileStepProps = {
  form: DocumentForm;
  documentProfiles: DocumentProfileItem[];
  selectedProfile: DocumentProfileItem | null;
  onDocumentFormChange: (next: DocumentForm) => void;
  onApplyProfile: (profileCode: string, preferredProcessArea?: string) => void | Promise<void>;
};

export function DocumentCreateProfileStep(props: DocumentCreateProfileStepProps) {
  const profileOptions: SelectMenuOption[] = props.documentProfiles.map((item) => ({
    value: item.code,
    label: metalNobreProfileOptionLabel(item),
  }));

  return (
    <div className="create-doc-identification-stack">
      <CreateField label="Titulo do documento" required>
        <input
          id="document-title"
          data-testid="document-title"
          placeholder="ex: Procedimento de atendimento ao cliente"
          value={props.form.title}
          onChange={(event) => props.onDocumentFormChange({ ...props.form, title: event.target.value })}
          required
        />
      </CreateField>

      <div className="field">
        <CreateField
          label="Tipo documental"
          required
          hint={props.selectedProfile ? metalNobreProfileContext(props.selectedProfile.code) : "Determine tipo canonico, schema e governanca do documento."}
        >
          <FilterDropdown
            id="document-profile"
            value={props.form.documentProfile}
            options={profileOptions}
            onSelect={(value) => void props.onApplyProfile(value, props.form.processArea)}
          />
        </CreateField>

        <div className="profile-preview">
          <span className={`profile-badge profile-${props.form.documentProfile}`}>{props.form.documentProfile.toUpperCase() || "--"}</span>
          <div>
            <div className="profile-name">{props.selectedProfile?.name ?? "Selecione um profile"}</div>
            <div className="profile-family">Family: {props.selectedProfile?.familyCode ?? "-"}</div>
          </div>
        </div>
      </div>
    </div>
  );
}
