import { useEffect, useState } from "react";
import { metalNobreProcessAreaHint, metalNobreProfileContext } from "../features/documents/adapters/metalNobreExperience";
import type { DocumentProfileGovernanceItem, DocumentProfileItem, DocumentProfileSchemaItem, ProcessAreaItem, SubjectItem } from "../lib.types";
import { WorkspaceDataState } from "./WorkspaceDataState";
import { WorkspaceViewFrame } from "./WorkspaceViewFrame";

type LoadState = "idle" | "loading" | "ready" | "error";

type RegistryExplorerProps = {
  loadState: LoadState;
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  subjects: SubjectItem[];
  selectedProfileCode: string;
  selectedProfileSchema: DocumentProfileSchemaItem | null;
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  showAdmin: boolean;
  onRefreshWorkspace: () => void | Promise<void>;
  onSelectProfile: (profileCode: string) => void | Promise<void>;
  onCreateProcessArea: (payload: { code: string; name: string; description: string }) => void | Promise<void>;
  onUpdateProcessArea: (payload: { code: string; name: string; description: string }) => void | Promise<void>;
  onDeleteProcessArea: (code: string) => void | Promise<void>;
  onCreateSubject: (payload: { code: string; processAreaCode: string; name: string; description: string }) => void | Promise<void>;
  onUpdateSubject: (payload: { code: string; processAreaCode: string; name: string; description: string }) => void | Promise<void>;
  onDeleteSubject: (code: string) => void | Promise<void>;
  onCreateDocumentProfile: (payload: { code: string; familyCode: string; name: string; alias: string; description: string; reviewIntervalDays: number }) => void | Promise<void>;
  onUpdateDocumentProfile: (payload: { code: string; familyCode: string; name: string; alias: string; description: string; reviewIntervalDays: number }) => void | Promise<void>;
  onDeleteDocumentProfile: (code: string) => void | Promise<void>;
  onUpdateDocumentProfileGovernance: (payload: { profileCode: string; workflowProfile: string; reviewIntervalDays: number; approvalRequired: boolean; retentionDays: number; validityDays: number }) => void | Promise<void>;
};

export function RegistryExplorer(props: RegistryExplorerProps) {
  const selectedProfile = props.documentProfiles.find((item) => item.code === props.selectedProfileCode) ?? props.documentProfiles[0] ?? null;
  const hasRegistryData = props.documentProfiles.length > 0;
  const [profileForm, setProfileForm] = useState({ code: "", familyCode: "", name: "", alias: "", description: "", reviewIntervalDays: 365 });
  const [governanceForm, setGovernanceForm] = useState({ workflowProfile: "standard_approval", reviewIntervalDays: 365, approvalRequired: true, retentionDays: 3650, validityDays: 0 });
  const [processAreaForm, setProcessAreaForm] = useState({ code: "", name: "", description: "" });
  const [subjectForm, setSubjectForm] = useState({ code: "", processAreaCode: "", name: "", description: "" });

  useEffect(() => {
    if (!selectedProfile) {
      return;
    }
    setProfileForm({
      code: selectedProfile.code,
      familyCode: selectedProfile.familyCode,
      name: selectedProfile.name,
      alias: selectedProfile.alias,
      description: selectedProfile.description,
      reviewIntervalDays: selectedProfile.reviewIntervalDays,
    });
    setGovernanceForm({
      workflowProfile: props.selectedProfileGovernance?.workflowProfile ?? "standard_approval",
      reviewIntervalDays: props.selectedProfileGovernance?.reviewIntervalDays ?? selectedProfile.reviewIntervalDays,
      approvalRequired: props.selectedProfileGovernance?.approvalRequired ?? true,
      retentionDays: props.selectedProfileGovernance?.retentionDays ?? 0,
      validityDays: props.selectedProfileGovernance?.validityDays ?? 0,
    });
  }, [props.selectedProfileGovernance, selectedProfile]);

  return (
    <WorkspaceViewFrame
      kicker="Registry explorer"
      title="Registry documental"
      description="Leitura operacional do motor profile-first aplicado ao contexto Metal Nobre, com foco em governanca e rastreabilidade."
    >
      <WorkspaceDataState
        loadState={props.loadState}
        isEmpty={!hasRegistryData}
        emptyTitle="Registry sem perfis configurados"
        emptyDescription="Nao existem perfis documentais ativos para consulta neste ambiente."
        loadingLabel="Atualizando registry documental"
        errorDescription="Nao foi possivel carregar perfis, schema e governanca agora."
        onRetry={props.onRefreshWorkspace}
      />

      {props.loadState === "ready" && hasRegistryData && (
      <div className="catalog-grid">
        <section className="catalog-panel catalog-list-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Profiles</p>
              <h2>Catalogo configurado</h2>
            </div>
          </div>
          <ul className="catalog-mini-list registry-list">
            {props.documentProfiles.map((item) => (
              <li key={item.code}>
                <button type="button" className={`registry-profile-button ${selectedProfile?.code === item.code ? "is-active" : ""}`} onClick={() => void props.onSelectProfile(item.code)}>
                  <span>{item.name}</span>
                  <small>{item.alias} / {item.code} / family {item.familyCode}</small>
                </button>
              </li>
            ))}
          </ul>
        </section>

        <aside className="catalog-panel catalog-detail-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Profile detail</p>
              <h2>{selectedProfile?.name ?? "Sem profile"}</h2>
            </div>
          </div>

          <div className="catalog-detail-stack">
            <div className="catalog-info-grid">
              <div><span>Code</span><strong>{selectedProfile?.code ?? "-"}</strong></div>
              <div><span>Alias</span><strong>{selectedProfile?.alias ?? "-"}</strong></div>
              <div><span>Family</span><strong>{selectedProfile?.familyCode ?? "-"}</strong></div>
              <div><span>Schema ativo</span><strong>{props.selectedProfileSchema ? `v${props.selectedProfileSchema.version}` : "-"}</strong></div>
              <div><span>Workflow</span><strong>{props.selectedProfileGovernance?.workflowProfile ?? "-"}</strong></div>
            </div>

            <div className="catalog-card">
              <h3>Schema metadata rules</h3>
              <ul className="catalog-mini-list">
                {(props.selectedProfileSchema?.metadataRules ?? []).map((rule) => (
                  <li key={rule.name}>
                    <span>{rule.name}</span>
                    <small>{rule.type}{rule.required ? " / required" : ""}</small>
                  </li>
                ))}
                {(props.selectedProfileSchema?.metadataRules ?? []).length === 0 && <li><span>Sem regras de metadata carregadas.</span></li>}
              </ul>
            </div>

            <div className="catalog-card">
              <h3>Governanca</h3>
              <ul className="catalog-mini-list">
                <li><span>Approval required</span><small>{props.selectedProfileGovernance?.approvalRequired ? "Sim" : "Nao"}</small></li>
                <li><span>Review interval</span><small>{props.selectedProfileGovernance ? `${props.selectedProfileGovernance.reviewIntervalDays} dias` : "-"}</small></li>
                <li><span>Retention</span><small>{props.selectedProfileGovernance?.retentionDays ? `${props.selectedProfileGovernance.retentionDays} dias` : "-"}</small></li>
                <li><span>Validity</span><small>{props.selectedProfileGovernance?.validityDays ? `${props.selectedProfileGovernance.validityDays} dias` : "-"}</small></li>
              </ul>
              {props.showAdmin && selectedProfile && (
                <div className="stack">
                  <input
                    placeholder="Workflow profile"
                    value={governanceForm.workflowProfile}
                    onChange={(event) => setGovernanceForm((current) => ({ ...current, workflowProfile: event.target.value }))}
                  />
                  <input
                    type="number"
                    min={1}
                    placeholder="Intervalo de revisao (dias)"
                    value={governanceForm.reviewIntervalDays}
                    onChange={(event) => setGovernanceForm((current) => ({ ...current, reviewIntervalDays: Number(event.target.value || 0) }))}
                  />
                  <input
                    type="number"
                    min={0}
                    placeholder="Retencao (dias)"
                    value={governanceForm.retentionDays}
                    onChange={(event) => setGovernanceForm((current) => ({ ...current, retentionDays: Number(event.target.value || 0) }))}
                  />
                  <input
                    type="number"
                    min={0}
                    placeholder="Validade (dias)"
                    value={governanceForm.validityDays}
                    onChange={(event) => setGovernanceForm((current) => ({ ...current, validityDays: Number(event.target.value || 0) }))}
                  />
                  <label>
                    <input
                      type="checkbox"
                      checked={governanceForm.approvalRequired}
                      onChange={(event) => setGovernanceForm((current) => ({ ...current, approvalRequired: event.target.checked }))}
                    />
                    Aprovacao obrigatoria
                  </label>
                  <div className="stack-inline">
                    <button
                      type="button"
                      onClick={() => void props.onUpdateDocumentProfileGovernance({
                        profileCode: selectedProfile.code,
                        workflowProfile: governanceForm.workflowProfile,
                        reviewIntervalDays: governanceForm.reviewIntervalDays,
                        approvalRequired: governanceForm.approvalRequired,
                        retentionDays: governanceForm.retentionDays,
                        validityDays: governanceForm.validityDays,
                      })}
                    >
                      Atualizar governanca
                    </button>
                  </div>
                </div>
              )}
            </div>

            <div className="catalog-card">
              <h3>Contexto aplicado</h3>
              <p className="catalog-muted">{selectedProfile ? metalNobreProfileContext(selectedProfile.code) : "Selecione um profile."}</p>
            </div>

            <div className="catalog-card">
              <h3>Profiles (admin)</h3>
              {props.showAdmin && (
                <div className="stack">
                  <input
                    placeholder="Codigo do profile"
                    value={profileForm.code}
                    onChange={(event) => setProfileForm((current) => ({ ...current, code: event.target.value }))}
                  />
                  <input
                    placeholder="Family code"
                    value={profileForm.familyCode}
                    onChange={(event) => setProfileForm((current) => ({ ...current, familyCode: event.target.value }))}
                  />
                  <input
                    placeholder="Nome do profile"
                    value={profileForm.name}
                    onChange={(event) => setProfileForm((current) => ({ ...current, name: event.target.value }))}
                  />
                  <input
                    placeholder="Alias curto"
                    value={profileForm.alias}
                    onChange={(event) => setProfileForm((current) => ({ ...current, alias: event.target.value }))}
                  />
                  <input
                    placeholder="Descricao"
                    value={profileForm.description}
                    onChange={(event) => setProfileForm((current) => ({ ...current, description: event.target.value }))}
                  />
                  <input
                    type="number"
                    min={1}
                    placeholder="Revisao (dias)"
                    value={profileForm.reviewIntervalDays}
                    onChange={(event) => setProfileForm((current) => ({ ...current, reviewIntervalDays: Number(event.target.value || 0) }))}
                  />
                  <div className="stack-inline">
                    <button type="button" onClick={() => void props.onCreateDocumentProfile(profileForm)}>Criar profile</button>
                    <button type="button" className="ghost-button" onClick={() => void props.onUpdateDocumentProfile(profileForm)}>Atualizar profile</button>
                    <button type="button" className="ghost-button" onClick={() => void props.onDeleteDocumentProfile(profileForm.code)}>Desativar profile</button>
                  </div>
                </div>
              )}
            </div>

            <div className="catalog-card">
              <h3>Process areas</h3>
              <ul className="catalog-mini-list">
                {props.processAreas.map((item) => <li key={item.code}><span>{item.name}</span><small>{metalNobreProcessAreaHint(item.code)}</small></li>)}
              </ul>
              {props.showAdmin && (
                <div className="stack">
                  <input
                    placeholder="Codigo da area"
                    value={processAreaForm.code}
                    onChange={(event) => setProcessAreaForm((current) => ({ ...current, code: event.target.value }))}
                  />
                  <input
                    placeholder="Nome da area"
                    value={processAreaForm.name}
                    onChange={(event) => setProcessAreaForm((current) => ({ ...current, name: event.target.value }))}
                  />
                  <input
                    placeholder="Descricao da area"
                    value={processAreaForm.description}
                    onChange={(event) => setProcessAreaForm((current) => ({ ...current, description: event.target.value }))}
                  />
                  <div className="stack-inline">
                    <button type="button" onClick={() => void props.onCreateProcessArea(processAreaForm)}>Criar area</button>
                    <button type="button" className="ghost-button" onClick={() => void props.onUpdateProcessArea(processAreaForm)}>Atualizar area</button>
                    <button type="button" className="ghost-button" onClick={() => void props.onDeleteProcessArea(processAreaForm.code)}>Desativar area</button>
                  </div>
                </div>
              )}
            </div>

            <div className="catalog-card">
              <h3>Subjects</h3>
              <ul className="catalog-mini-list">
                {props.subjects.map((item) => <li key={item.code}><span>{item.name}</span><small>{item.processAreaCode}</small></li>)}
                {props.subjects.length === 0 && <li><span>Sem subjects retornados.</span></li>}
              </ul>
              {props.showAdmin && (
                <div className="stack">
                  <input
                    placeholder="Codigo do subject"
                    value={subjectForm.code}
                    onChange={(event) => setSubjectForm((current) => ({ ...current, code: event.target.value }))}
                  />
                  <select
                    value={subjectForm.processAreaCode}
                    onChange={(event) => setSubjectForm((current) => ({ ...current, processAreaCode: event.target.value }))}
                  >
                    <option value="">Selecione a area</option>
                    {props.processAreas.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}
                  </select>
                  <input
                    placeholder="Nome do subject"
                    value={subjectForm.name}
                    onChange={(event) => setSubjectForm((current) => ({ ...current, name: event.target.value }))}
                  />
                  <input
                    placeholder="Descricao do subject"
                    value={subjectForm.description}
                    onChange={(event) => setSubjectForm((current) => ({ ...current, description: event.target.value }))}
                  />
                  <div className="stack-inline">
                    <button type="button" onClick={() => void props.onCreateSubject(subjectForm)}>Criar subject</button>
                    <button type="button" className="ghost-button" onClick={() => void props.onUpdateSubject(subjectForm)}>Atualizar subject</button>
                    <button type="button" className="ghost-button" onClick={() => void props.onDeleteSubject(subjectForm.code)}>Desativar subject</button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </aside>
      </div>
      )}
    </WorkspaceViewFrame>
  );
}
