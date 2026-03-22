import type { ReactNode } from "react";
import type { WizardStep } from "../documentCreateTypes";
import { ProgressSidebar, type ProgressSidebarItem } from "./ProgressSidebar";

type CreateDocumentShellProps = {
  steps: Array<{ key: WizardStep; label: string; description: string }>;
  currentStep: WizardStep;
  stepStates: Record<WizardStep, "pending" | "active" | "done" | "error">;
  onStepChange: (step: WizardStep) => void;
  children: ReactNode;
};

export function CreateDocumentShell(props: CreateDocumentShellProps) {
  const items: ProgressSidebarItem[] = props.steps.map((item, index) => ({
    key: item.key,
    label: item.label,
    description: item.description,
    status: props.stepStates[item.key],
    isCurrent: props.currentStep === item.key,
    onSelect: () => props.onStepChange(item.key),
  }));

  return (
    <div className="create-doc-layout">
      <ProgressSidebar items={items} />
      <div className="create-doc-main">{props.children}</div>
    </div>
  );
}
