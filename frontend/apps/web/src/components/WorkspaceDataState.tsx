type LoadState = "idle" | "loading" | "ready" | "error";

type WorkspaceDataStateProps = {
  loadState: LoadState;
  isEmpty: boolean;
  emptyTitle: string;
  emptyDescription: string;
  loadingLabel?: string;
  errorTitle?: string;
  errorDescription?: string;
  onRetry?: () => void | Promise<void>;
};

export function WorkspaceDataState(props: WorkspaceDataStateProps) {
  const onRetry = props.onRetry;

  if (props.loadState === "loading") {
    return (
      <section className="catalog-panel placeholder-panel">
        <div className="catalog-panel-head">
          <div>
            <p className="catalog-kicker">Carregando</p>
            <h2>{props.loadingLabel ?? "Atualizando workspace"}</h2>
          </div>
        </div>
        <p className="catalog-muted">Aguarde um instante enquanto sincronizamos os dados mais recentes.</p>
      </section>
    );
  }

  if (props.loadState === "error") {
    return (
      <section className="catalog-panel placeholder-panel">
        <div className="catalog-panel-head">
          <div>
            <p className="catalog-kicker">Falha operacional</p>
            <h2>{props.errorTitle ?? "Nao foi possivel carregar os dados"}</h2>
          </div>
        </div>
        <p className="catalog-muted">{props.errorDescription ?? "Tente novamente para sincronizar o workspace."}</p>
        {onRetry && (
          <button type="button" className="ghost-button" onClick={() => void onRetry()}>
            Tentar novamente
          </button>
        )}
      </section>
    );
  }

  if (!props.isEmpty) {
    return null;
  }

  return (
    <section className="catalog-panel placeholder-panel">
      <div className="catalog-panel-head">
        <div>
          <p className="catalog-kicker">Sem dados</p>
          <h2>{props.emptyTitle}</h2>
        </div>
      </div>
      <p className="catalog-muted">{props.emptyDescription}</p>
    </section>
  );
}
