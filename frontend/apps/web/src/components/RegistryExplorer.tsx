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
  onRefreshWorkspace: () => void | Promise<void>;
  onSelectProfile: (profileCode: string) => void | Promise<void>;
};

export function RegistryExplorer(props: RegistryExplorerProps) {
  const selectedProfile = props.documentProfiles.find((item) => item.code === props.selectedProfileCode) ?? props.documentProfiles[0] ?? null;
  const hasRegistryData = props.documentProfiles.length > 0;

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
            </div>

            <div className="catalog-card">
              <h3>Contexto aplicado</h3>
              <p className="catalog-muted">{selectedProfile ? metalNobreProfileContext(selectedProfile.code) : "Selecione um profile."}</p>
            </div>

            <div className="catalog-card">
              <h3>Process areas</h3>
              <ul className="catalog-mini-list">
                {props.processAreas.map((item) => <li key={item.code}><span>{item.name}</span><small>{metalNobreProcessAreaHint(item.code)}</small></li>)}
              </ul>
            </div>

            <div className="catalog-card">
              <h3>Subjects</h3>
              <ul className="catalog-mini-list">
                {props.subjects.map((item) => <li key={item.code}><span>{item.name}</span><small>{item.processAreaCode}</small></li>)}
                {props.subjects.length === 0 && <li><span>Sem subjects retornados.</span></li>}
              </ul>
            </div>
          </div>
        </aside>
      </div>
      )}
    </WorkspaceViewFrame>
  );
}
