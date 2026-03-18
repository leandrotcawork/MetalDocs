type WorkspacePlaceholderProps = {
  title: string;
  kicker: string;
  description: string;
  bullets: string[];
};

export function WorkspacePlaceholder(props: WorkspacePlaceholderProps) {
  return (
    <section className="catalog-shell">
      <div className="catalog-header">
        <div>
          <p className="catalog-kicker">{props.kicker}</p>
          <h1>{props.title}</h1>
          <p>{props.description}</p>
        </div>
      </div>
      <div className="catalog-grid single">
        <section className="catalog-panel placeholder-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Planejado</p>
              <h2>Pronto para a proxima iteracao</h2>
            </div>
          </div>
          <ul className="placeholder-list">
            {props.bullets.map((bullet) => <li key={bullet}>{bullet}</li>)}
          </ul>
        </section>
      </div>
    </section>
  );
}
