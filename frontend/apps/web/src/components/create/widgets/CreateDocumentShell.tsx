import type { ReactNode } from "react";
import type { StepStatus, WizardStep } from "../documentCreateTypes";

type CreateDocumentShellProps = {
  steps: Array<{ key: WizardStep; label: string; description: string }>;
  currentStep: WizardStep;
  stepStates: Record<WizardStep, StepStatus>;
  onStepChange: (step: WizardStep) => void;
  children: ReactNode;
};

function glyphFor(status: StepStatus, index: number): ReactNode {
  if (status === "done") {
    return (
      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M3 8.5 6.3 11.5 13 4.8" />
      </svg>
    );
  }
  if (status === "error") {
    return (
      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M4.2 4.2 11.8 11.8M11.8 4.2 4.2 11.8" />
      </svg>
    );
  }
  return index + 1;
}

export function CreateDocumentShell(props: CreateDocumentShellProps) {
  return (
    <div className="create-doc-layout">
      <aside className="create-doc-steps">
        <p className="create-doc-steps-title">Etapas</p>
        <div className="create-doc-steps-list">
          {props.steps.map((item, index) => (
            <div key={item.key} className="create-doc-step-row">
              <button
                type="button"
                className="create-doc-step-item"
                data-status={props.stepStates[item.key]}
                aria-current={props.currentStep === item.key ? "step" : undefined}
                onClick={() => props.onStepChange(item.key)}
              >
                <span className="create-doc-step-num">
                  <span className="create-doc-step-glyph">
                    {glyphFor(props.stepStates[item.key], index)}
                  </span>
                </span>
                <div>
                  <strong>{item.label}</strong>
                  <small>{item.description}</small>
                </div>
              </button>
              {index < props.steps.length - 1 && (
                <div className="create-doc-step-connector" data-status={props.stepStates[item.key]} aria-hidden="true" />
              )}
            </div>
          ))}
        </div>
      </aside>

      <div className="create-doc-main">{props.children}</div>
    </div>
  );
}

