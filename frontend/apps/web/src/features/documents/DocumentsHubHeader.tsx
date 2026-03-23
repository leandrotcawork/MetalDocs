import { SearchBar } from "../../components/ui/SearchBar";
import styles from "./DocumentsHubView.module.css";

type DocumentsHubHeaderProps = {
  title: string;
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
};

export function DocumentsHubHeader(props: DocumentsHubHeaderProps) {
  return (
    <header className={styles.pageHeader}>
      <div className={styles.hero}>
        <div className={styles.heroCopy}>
          <h1 className={styles.title}>{props.title}</h1>
          <p className={styles.subtitle}>
            Acervo organizado por areas, tipos e status. Navegue pelos documentos mais relevantes.
          </p>
        </div>
        <div className={styles.headerSearch}>
          <SearchBar value={props.searchQuery} onChange={props.onSearchQueryChange} />
        </div>
      </div>
    </header>
  );
}
