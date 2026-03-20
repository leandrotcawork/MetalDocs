import { useMemo, useState } from "react";
import { DocumentCreateContentStep } from "./create/DocumentCreateContentStep";
import { DocumentCreateBodyStep } from "./create/DocumentCreateBodyStep";
import { DocumentCreateContextStep } from "./create/DocumentCreateContextStep";
import { DocumentCreateMetadataStep } from "./create/DocumentCreateMetadataStep";
import { DocumentCreateProfileStep } from "./create/DocumentCreateProfileStep";
import type { DocumentCreateViewProps } from "./create/documentCreateTypes";
import { parseMetadata, wizardSteps } from "./create/documentCreateTypes";
import { CreateDocumentSection } from "./create/widgets/CreateDocumentSection";
import { CreateDocumentShell } from "./create/widgets/CreateDocumentShell";
import type { StepStatus, WizardStep } from "./create/documentCreateTypes";

export function DocumentCreateView(props: DocumentCreateViewProps) {
  const selectedProfile = props.documentProfiles.find((item) => item.code === props.documentForm.documentProfile) ?? null;
  const [currentStep, setCurrentStep] = useState<WizardStep>("identification");
  const [maxVisitedIndex, setMaxVisitedIndex] = useState(0);
  const stepIndexByKey = useMemo(
    () => wizardSteps.reduce<Record<WizardStep, number>>((acc, step, index) => {
      acc[step.key] = index;
      return acc;
    }, {} as Record<WizardStep, number>),
    [],
  );
  const sectionIdByStep: Record<WizardStep, string> = {
    identification: "create-section-identification",
    context: "create-section-context",
    metadata: "create-section-metadata",
    content: "create-section-content",
    body: "create-section-body",
  };
  const metadataRules = props.selectedProfileSchema?.metadataRules ?? [];
  const metadataMap = parseMetadata(props.documentForm.metadata);
  const metadataComplete = metadataRules.length === 0
    ? true
    : metadataRules.every((rule) => !rule.required || (metadataMap[rule.name]?.toString().trim() ?? "") !== "");
  const isConfidential = props.documentForm.classification === "CONFIDENTIAL";
  const isRestricted = props.documentForm.classification === "RESTRICTED";
  const requiresAudience = isConfidential || isRestricted;
  const audienceComplete = !requiresAudience || (
    isConfidential
      ? props.documentForm.audienceDepartments.length > 0
      : props.documentForm.audienceDepartment.trim().length > 0 && props.documentForm.audienceProcessArea.trim().length > 0
  );
  const contentCompleted = props.contentMode === "native"
    ? true
    : Boolean(props.contentFile || props.contentPdfUrl);
  const stepCompletion: Record<WizardStep, boolean> = {
    identification: props.documentForm.title.trim().length > 0 && props.documentForm.documentProfile.trim().length > 0,
    context: props.documentForm.ownerId.trim().length > 0
      && props.documentForm.businessUnit.trim().length > 0
      && props.documentForm.department.trim().length > 0,
    metadata: metadataComplete,
    content: props.documentForm.classification.trim().length > 0
      && audienceComplete,
    body: contentCompleted,
  };
  function stepStateFor(step: WizardStep): StepStatus {
    if (step === currentStep) {
      return "active";
    }
    if (stepCompletion[step]) {
      return "done";
    }
    return stepIndexByKey[step] < maxVisitedIndex ? "error" : "pending";
  }

  function handleStepChange(step: WizardStep) {
    const nextIndex = stepIndexByKey[step];
    setCurrentStep(step);
    setMaxVisitedIndex((current) => Math.max(current, nextIndex));
    const targetId = sectionIdByStep[step];
    window.requestAnimationFrame(() => {
      const target = document.getElementById(targetId);
      if (target) {
        target.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    });
  }

  const stepStates = useMemo(
    () => wizardSteps.reduce<Record<WizardStep, StepStatus>>((acc, step) => {
      acc[step.key] = stepStateFor(step.key);
      return acc;
    }, {} as Record<WizardStep, StepStatus>),
    [currentStep, maxVisitedIndex, stepCompletion, stepIndexByKey],
  );

  return (
    <CreateDocumentShell
      steps={wizardSteps}
      currentStep={currentStep}
      stepStates={stepStates}
      onStepChange={handleStepChange}
    >
      <div className="create-doc-toolbar">
        <span className="create-doc-breadcrumb-link">MetalDocs</span>
        <span className="create-doc-breadcrumb-sep">/</span>
        <span className="create-doc-breadcrumb-link">Acervo</span>
        <span className="create-doc-breadcrumb-sep">/</span>
        <strong>Novo documento</strong>
      </div>

      <form data-testid="document-create-form" className="create-doc-content" onSubmit={props.onSubmitCreateDocument}>
        <div className="create-doc-form-wrap">
          <div className="create-doc-page-title">
            <h2>Criar documento</h2>
            <p>Campos marcados com * sao obrigatorios.</p>
          </div>

          <CreateDocumentSection
            sectionId={sectionIdByStep.identification}
            title="Identificacao documental"
            subtitle="Defina titulo e profile canonico do documento."
            icon={
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round">
                <path d="M2 2h10v10H2z" strokeLinejoin="round" />
                <path d="M5 5.5h4M5 8h4M5 10.5h2" />
              </svg>
            }
          >
            <DocumentCreateProfileStep
              form={props.documentForm}
              documentProfiles={props.documentProfiles}
              selectedProfile={selectedProfile}
              onDocumentFormChange={props.onDocumentFormChange}
              onApplyProfile={props.onApplyProfile}
            />
          </CreateDocumentSection>

          <CreateDocumentSection
            sectionId={sectionIdByStep.context}
            title="Contexto operacional"
            subtitle="Responsavel, unidade, departamento e taxonomia."
            icon={
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round">
                <circle cx="7" cy="4.5" r="2.3" />
                <path d="M2.5 12c0-2.4 2-4.3 4.5-4.3s4.5 1.9 4.5 4.3" />
              </svg>
            }
          >
            <DocumentCreateContextStep
              form={props.documentForm}
              processAreas={props.processAreas}
              documentDepartments={props.documentDepartments}
              subjects={props.subjects}
              onDocumentFormChange={props.onDocumentFormChange}
            />
          </CreateDocumentSection>

          <CreateDocumentSection
            sectionId={sectionIdByStep.metadata}
            title="Metadata dinamica"
            subtitle="Campos do schema ativo, sem hardcode local."
            icon={
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round">
                <path d="M2.5 4h9M2.5 7h9M2.5 10h9" />
              </svg>
            }
          >
            <DocumentCreateMetadataStep
              form={props.documentForm}
              selectedProfileSchema={props.selectedProfileSchema}
              onDocumentFormChange={props.onDocumentFormChange}
            />
          </CreateDocumentSection>

          <CreateDocumentSection
            sectionId={sectionIdByStep.content}
            title="Classificacao e acesso"
            subtitle="Classificacao, audiencia e vigencia."
            icon={
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round">
                <path d="M3 2h6.5L12 4.5V12H3V2z" strokeLinejoin="round" />
                <path d="M9.5 2v2.5H12" strokeLinejoin="round" />
              </svg>
            }
          >
            <DocumentCreateContentStep
              form={props.documentForm}
              processAreas={props.processAreas}
              documentDepartments={props.documentDepartments}
              onDocumentFormChange={props.onDocumentFormChange}
            />
          </CreateDocumentSection>

          <CreateDocumentSection
            sectionId={sectionIdByStep.body}
            title="Conteudo do documento"
            subtitle="Escolha como voce vai produzir o conteudo."
            icon={(
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round">
                <path d="M3 3.5h8M3 6.5h8M3 9.5h5" />
              </svg>
            )}
          >
          <DocumentCreateBodyStep
            contentMode={props.contentMode}
            contentFile={props.contentFile}
            contentPdfUrl={props.contentPdfUrl}
            contentDocxUrl={props.contentDocxUrl}
            contentStatus={props.contentStatus}
            contentError={props.contentError}
            profileCode={selectedProfile?.code ?? ""}
            onContentModeChange={props.onContentModeChange}
            onContentFileChange={props.onContentFileChange}
            onDownloadTemplate={props.onDownloadTemplate}
          />
          </CreateDocumentSection>

                    <footer className="create-doc-footer">
            <button type="button" className="ghost-button create-doc-footer-back">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M10.5 3.5L6 8l4.5 4.5" />
              </svg>
              Voltar
            </button>
            <div className="create-doc-footer-actions">
              <button data-testid="document-submit" type="submit" className="create-doc-footer-primary" disabled={props.isSubmitting}>
                <svg viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
                  <path d="M2 7l3.5 3.5L11 4" />
                </svg>
                {props.isSubmitting ? "Abrindo editor..." : "Salvar e ir para o editor"}
              </button>
            </div>
          </footer>
        </div>
      </form>
      {props.isSubmitting && (
        <div className="create-doc-transition" role="status" aria-live="polite">
          <div className="create-doc-transition-card">
            <div className="create-doc-transition-title">Abrindo editor</div>
            <p className="create-doc-transition-copy">Preparando schema, versoes e preview do documento.</p>
            <div className="create-doc-transition-skeleton">
              <div className="create-doc-transition-line is-wide" />
              <div className="create-doc-transition-line" />
              <div className="create-doc-transition-line is-short" />
            </div>
          </div>
        </div>
      )}
    </CreateDocumentShell>
  );
}

