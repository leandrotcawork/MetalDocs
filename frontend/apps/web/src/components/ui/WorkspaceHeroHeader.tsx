import { SearchBar } from "./SearchBar";
import styles from "./WorkspaceHeroHeader.module.css";

type WorkspaceHeroHeaderProps = {
  title: string;
  subtitle?: string;
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  variant?: "default" | "compact";
};

export function WorkspaceHeroHeader(props: WorkspaceHeroHeaderProps) {
  const isCompact = props.variant === "compact";
  const headerClassName = `${styles.header} ${isCompact ? styles.headerCompact : ""}`.trim();
  const heroClassName = `${styles.hero} ${isCompact ? styles.heroCompact : ""}`.trim();
  const titleClassName = `${styles.title} ${isCompact ? styles.titleCompact : ""}`.trim();

  return (
    <header className={headerClassName}>
      <div className={heroClassName}>
        <div className={styles.copy}>
          <h1 className={titleClassName}>{props.title}</h1>
          {!isCompact && props.subtitle && <p className={styles.subtitle}>{props.subtitle}</p>}
        </div>
        <div className={styles.search}>
          <SearchBar value={props.searchQuery} onChange={props.onSearchQueryChange} />
        </div>
      </div>
    </header>
  );
}
