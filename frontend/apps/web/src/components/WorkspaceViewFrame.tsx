import type { ReactNode } from "react";

type WorkspaceViewFrameProps = {
  kicker: string;
  title: string;
  description: string;
  actions?: ReactNode;
  stats?: ReactNode;
  className?: string;
  testId?: string;
  children: ReactNode;
};

export function WorkspaceViewFrame(props: WorkspaceViewFrameProps) {
  const className = props.className ? `catalog-shell ${props.className}` : "catalog-shell";
  return (
    <section data-testid={props.testId} className={className}>
      <div className="catalog-header">
        <div>
          <p className="catalog-kicker">{props.kicker}</p>
          <h1>{props.title}</h1>
          <p>{props.description}</p>
        </div>
        {props.actions}
      </div>
      {props.stats}
      {props.children}
    </section>
  );
}
