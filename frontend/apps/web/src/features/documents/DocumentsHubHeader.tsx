import { SearchBar } from "../../components/ui/SearchBar";
import styles from "./DocumentsHubView.module.css";

type DocumentsHubHeaderProps = {
  title: string;
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  variant?: "default" | "compact";
};

export function DocumentsHubHeader(props: DocumentsHubHeaderProps) {
  const isCompact = props.variant === "compact";
  const variantClass = isCompact ? styles.pageHeaderCompact : "";
  const heroClass = isCompact ? `${styles.hero} ${styles.heroCompact}` : styles.hero;
  const titleClass = isCompact ? `${styles.title} ${styles.titleCompact}` : styles.title;
  return (
    <header className={`${styles.pageHeader} ${variantClass}`.trim()}>
      <div className={heroClass}>
        <div className={styles.heroCopy}>
          <h1 className={titleClass}>{props.title}</h1>
          {!isCompact && (
            <p className={styles.subtitle}>
              Acervo organizado por areas, tipos e status. Navegue pelos documentos mais relevantes.
            </p>
          )}
        </div>
        <div className={styles.headerSearch}>
          <SearchBar value={props.searchQuery} onChange={props.onSearchQueryChange} />
        </div>
      </div>
    </header>
  );
}
