import { useMemo, useState } from "react";
import styles from "./DashboardAlertsModal.module.css";

type DashboardAlertTone = "violet" | "orange" | "blue" | "gold" | "rose" | "indigo" | "green";

export type DashboardAlertItem = {
  id: string;
  title: string;
  skuCount: number;
  tone: DashboardAlertTone;
};

type DashboardAlertsModalProps = {
  open: boolean;
  totalAlerts: number;
  alerts: DashboardAlertItem[];
  onClose: () => void;
};

const nextSteps = [
  "Selecione um alerta para ver os SKUs impactados.",
  "Defina prioridade e responsavel por alerta.",
  "Execute a mitigacao no fluxo operacional.",
];

export function DashboardAlertsModal(props: DashboardAlertsModalProps) {
  const [query, setQuery] = useState("");
  const [selectedAlertId, setSelectedAlertId] = useState<string | null>(null);

  const visibleAlerts = useMemo(() => {
    const search = query.trim().toLowerCase();
    if (!search) return props.alerts;
    return props.alerts.filter((item) => item.title.toLowerCase().includes(search));
  }, [props.alerts, query]);

  const selectedAlert = useMemo(
    () => visibleAlerts.find((item) => item.id === selectedAlertId) ?? visibleAlerts[0] ?? null,
    [selectedAlertId, visibleAlerts],
  );

  if (!props.open) return null;

  return (
    <div className={styles.overlay} role="presentation" onClick={props.onClose}>
      <section className={styles.modal} role="dialog" aria-modal="true" aria-label="Alertas ativos" onClick={(event) => event.stopPropagation()}>
        <header className={styles.header}>
          <div>
            <h2>Alertas ativos</h2>
            <p>{props.totalAlerts} alertas mapeados</p>
          </div>
          <button type="button" className={styles.closeButton} aria-label="Fechar alertas" onClick={props.onClose}>x</button>
        </header>

        <article className={styles.card}>
          <h3>Alertas disponiveis</h3>
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className={styles.search}
            placeholder="Buscar no spotlight..."
            aria-label="Buscar alerta"
          />
          <div className={styles.alertGrid}>
            {visibleAlerts.map((item) => (
              <button
                key={item.id}
                type="button"
                className={`${styles.alertItem} ${styles[item.tone]} ${selectedAlert?.id === item.id ? styles.alertItemSelected : ""}`}
                onClick={() => setSelectedAlertId(item.id)}
              >
                <strong>{item.title}</strong>
                <span>{item.skuCount} SKUs</span>
              </button>
            ))}
          </div>
        </article>

        <article className={styles.card}>
          <h3>Proximos passos</h3>
          <div className={styles.steps}>
            {nextSteps.map((step, index) => (
              <div key={step} className={styles.stepRow}>
                <span className={styles.stepIndex}>{index + 1}</span>
                <span>{step}</span>
              </div>
            ))}
          </div>
        </article>
      </section>
    </div>
  );
}
