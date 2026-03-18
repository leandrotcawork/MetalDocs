import { useMemo, useState } from "react";
import type { DocumentProfileGovernanceItem, DocumentProfileItem, DocumentProfileSchemaItem, ProcessAreaItem, SubjectItem } from "../lib.types";

type DocumentForm = {
  title: string;
  documentType: string;
  documentProfile: string;
  processArea: string;
  subject: string;
  ownerId: string;
  businessUnit: string;
  department: string;
  classification: string;
  tags: string;
  metadata: string;
  initialContent: string;
};

type WizardStep = "profile" | "metadata" | "content" | "review";

type DocumentCreateViewProps = {
  documentForm: DocumentForm;
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  subjects: SubjectItem[];
  selectedProfileSchema: DocumentProfileSchemaItem | null;
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  onDocumentFormChange: (next: DocumentForm) => void;
  onApplyProfile: (profileCode: string, preferredProcessArea?: string) => void | Promise<void>;
  onSubmitCreateDocument: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
};

const wizardSteps: Array<{ key: WizardStep; label: string; description: string }> = [
  { key: "profile", label: "Profile", description: "Escolha o modelo documental e contexto operacional." },
  { key: "metadata", label: "Metadata", description: "Preencha campos guiados pelo schema ativo." },
  { key: "content", label: "Content", description: "Registre o texto base da versao inicial." },
  { key: "review", label: "Review", description: "Valide governanca, metadata e payload antes de criar." },
];

function parseMetadata(value: string): Record<string, string> {
  try {
    const parsed = JSON.parse(value);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    return Object.entries(parsed).reduce<Record<string, string>>((acc, [key, item]) => {
      acc[key] = typeof item === "string" ? item : JSON.stringify(item);
      return acc;
    }, {});
  } catch {
    return {};
  }
}

function updateMetadataField(source: string, key: string, nextValue: string): string {
  const metadata = parseMetadata(source);
  metadata[key] = nextValue;
  return JSON.stringify(metadata, null, 2);
}

export function DocumentCreateView(props: DocumentCreateViewProps) {
  const [step, setStep] = useState<WizardStep>("profile");

  const selectedProfile = props.documentProfiles.find((item) => item.code === props.documentForm.documentProfile) ?? null;
  const metadataMap = useMemo(() => parseMetadata(props.documentForm.metadata), [props.documentForm.metadata]);
  const availableSubjects = useMemo(
    () => props.subjects.filter((item) => !props.documentForm.processArea || item.processAreaCode === props.documentForm.processArea),
    [props.documentForm.processArea, props.subjects],
  );

  function moveStep(direction: 1 | -1) {
    const currentIndex = wizardSteps.findIndex((item) => item.key === step);
    const nextIndex = currentIndex + direction;
    if (nextIndex < 0 || nextIndex >= wizardSteps.length) {
      return;
    }
    setStep(wizardSteps[nextIndex].key);
  }

  return (
    <section className="catalog-shell">
      <div className="catalog-header">
        <div>
          <p className="catalog-kicker">Authoring</p>
          <h1>Document authoring wizard</h1>
          <p>Fluxo profile-first em quatro etapas, com governanca sempre visivel e metadata dinamica baseada no schema ativo.</p>
        </div>
      </div>

      <div className="wizard-layout">
        <aside className="catalog-panel wizard-sidebar">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Etapas</p>
              <h2>Profile to review</h2>
            </div>
          </div>
          <div className="wizard-step-list">
            {wizardSteps.map((item, index) => (
              <button
                key={item.key}
                type="button"
                className={`wizard-step-card ${step === item.key ? "is-active" : ""}`}
                onClick={() => setStep(item.key)}
              >
                <span className="wizard-step-index">0{index + 1}</span>
                <strong>{item.label}</strong>
                <small>{item.description}</small>
              </button>
            ))}
          </div>

          <div className="catalog-card">
            <h3>Governanca ativa</h3>
            <ul className="catalog-mini-list">
              <li><span>Workflow</span><small>{props.selectedProfileGovernance?.workflowProfile ?? "-"}</small></li>
              <li><span>Revisao</span><small>{props.selectedProfileGovernance ? `${props.selectedProfileGovernance.reviewIntervalDays} dias` : "-"}</small></li>
              <li><span>Aprovacao</span><small>{props.selectedProfileGovernance?.approvalRequired ? "Obrigatoria" : "Opcional"}</small></li>
              <li><span>Retencao</span><small>{props.selectedProfileGovernance?.retentionDays ? `${props.selectedProfileGovernance.retentionDays} dias` : "-"}</small></li>
            </ul>
          </div>
        </aside>

        <form data-testid="document-create-form" className="catalog-panel wizard-panel stack" onSubmit={props.onSubmitCreateDocument}>
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Etapa atual</p>
              <h2>{wizardSteps.find((item) => item.key === step)?.label}</h2>
            </div>
          </div>

          {step === "profile" && (
            <div className="stack">
              <div className="catalog-form-grid">
                <div>
                  <label htmlFor="document-title"><span>Titulo</span></label>
                  <input id="document-title" data-testid="document-title" placeholder="Titulo do documento" value={props.documentForm.title} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, title: event.target.value })} required />
                </div>
                <div>
                  <label htmlFor="document-profile"><span>Profile</span></label>
                  <select id="document-profile" data-testid="document-profile" value={props.documentForm.documentProfile} onChange={(event) => void props.onApplyProfile(event.target.value, props.documentForm.processArea)}>
                    {props.documentProfiles.map((item) => <option key={item.code} value={item.code}>{item.name} ({item.alias})</option>)}
                  </select>
                </div>
              </div>

              <div className="catalog-stats compact">
                <article className="catalog-stat"><span>Family derivada</span><strong>{selectedProfile?.familyCode ?? "-"}</strong></article>
                <article className="catalog-stat"><span>Schema ativo</span><strong>{props.selectedProfileSchema ? `v${props.selectedProfileSchema.version}` : "-"}</strong></article>
                <article className="catalog-stat"><span>Owner</span><strong>{props.documentForm.ownerId || "-"}</strong></article>
                <article className="catalog-stat"><span>Classificacao</span><strong>{props.documentForm.classification}</strong></article>
              </div>

              <div className="catalog-form-grid">
                <div>
                  <label htmlFor="document-process-area"><span>Process area</span></label>
                  <select id="document-process-area" data-testid="document-process-area" value={props.documentForm.processArea} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, processArea: event.target.value, subject: "" })}>
                    <option value="">Sem process area</option>
                    {props.processAreas.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="document-subject"><span>Subject</span></label>
                  <select id="document-subject" data-testid="document-subject" value={props.documentForm.subject} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, subject: event.target.value })}>
                    <option value="">Sem subject</option>
                    {availableSubjects.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="document-business-unit"><span>Business unit</span></label>
                  <input id="document-business-unit" data-testid="document-business-unit" placeholder="Business unit" value={props.documentForm.businessUnit} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, businessUnit: event.target.value })} />
                </div>
                <div>
                  <label htmlFor="document-department"><span>Departamento</span></label>
                  <input id="document-department" data-testid="document-department" placeholder="Departamento" value={props.documentForm.department} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, department: event.target.value })} />
                </div>
              </div>
            </div>
          )}

          {step === "metadata" && (
            <div className="stack">
              <p className="catalog-muted">Os campos abaixo nascem do schema ativo do profile. O JSON bruto continua visivel para nao esconder a fonte do payload.</p>
              <div className="metadata-rule-grid">
                {(props.selectedProfileSchema?.metadataRules ?? []).map((rule) => (
                  <div key={rule.name} className="metadata-rule-card">
                    <label htmlFor={`metadata-${rule.name}`}><span>{rule.name}</span></label>
                    <input
                      id={`metadata-${rule.name}`}
                      value={metadataMap[rule.name] ?? ""}
                      placeholder={rule.required ? "Obrigatorio" : "Opcional"}
                      onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, metadata: updateMetadataField(props.documentForm.metadata, rule.name, event.target.value) })}
                    />
                    <small>{rule.type}{rule.required ? " / required" : ""}</small>
                  </div>
                ))}
                {(props.selectedProfileSchema?.metadataRules ?? []).length === 0 && <p className="catalog-muted">O schema ativo nao trouxe regras de metadata.</p>}
              </div>
              <textarea data-testid="document-metadata" rows={12} value={props.documentForm.metadata} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, metadata: event.target.value })} />
            </div>
          )}

          {step === "content" && (
            <div className="stack">
              <div className="catalog-form-grid">
                <div>
                  <label htmlFor="document-owner"><span>Owner</span></label>
                  <input id="document-owner" data-testid="document-owner" placeholder="Owner" value={props.documentForm.ownerId} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, ownerId: event.target.value })} />
                </div>
                <div>
                  <label htmlFor="document-tags"><span>Tags</span></label>
                  <input id="document-tags" data-testid="document-tags" placeholder="Tags separadas por virgula" value={props.documentForm.tags} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, tags: event.target.value })} />
                </div>
              </div>
              <textarea data-testid="document-initial-content" rows={16} value={props.documentForm.initialContent} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, initialContent: event.target.value })} placeholder="Conteudo inicial da versao 1" />
            </div>
          )}

          {step === "review" && (
            <div className="stack">
              <div className="catalog-info-grid">
                <div><span>Profile</span><strong>{props.documentForm.documentProfile || "-"}</strong></div>
                <div><span>Family</span><strong>{selectedProfile?.familyCode ?? "-"}</strong></div>
                <div><span>Processo</span><strong>{props.documentForm.processArea || "-"}</strong></div>
                <div><span>Subject</span><strong>{props.documentForm.subject || "-"}</strong></div>
              </div>
              <div className="catalog-info-grid">
                <div><span>Owner</span><strong>{props.documentForm.ownerId || "-"}</strong></div>
                <div><span>Business unit</span><strong>{props.documentForm.businessUnit || "-"}</strong></div>
                <div><span>Departamento</span><strong>{props.documentForm.department || "-"}</strong></div>
                <div><span>Tags</span><strong>{props.documentForm.tags || "-"}</strong></div>
              </div>
              <div className="catalog-card">
                <h3>Metadata final</h3>
                <pre className="catalog-code-block">{props.documentForm.metadata}</pre>
              </div>
              <div className="catalog-card">
                <h3>Conteudo inicial</h3>
                <pre className="catalog-code-block">{props.documentForm.initialContent || "Sem conteudo inicial informado."}</pre>
              </div>
            </div>
          )}

          <div className="wizard-actions">
            <button type="button" className="ghost-button" onClick={() => moveStep(-1)} disabled={step === "profile"}>Voltar</button>
            {step === "review" ? (
              <button data-testid="document-submit" type="submit">Criar documento</button>
            ) : (
              <button type="button" onClick={() => moveStep(1)}>Continuar</button>
            )}
          </div>
        </form>
      </div>
    </section>
  );
}
