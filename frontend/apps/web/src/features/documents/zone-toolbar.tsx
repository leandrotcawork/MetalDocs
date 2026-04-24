import type { ContentPolicy } from '../templates/placeholder-types';

interface ZoneToolbarProps {
  zoneId: string;
  contentPolicy: ContentPolicy;
  onInsertTable?: () => void;
  onInsertImage?: () => void;
  onInsertHeading?: () => void;
  onInsertList?: () => void;
}

export function ZoneToolbar({
  zoneId,
  contentPolicy,
  onInsertTable,
  onInsertImage,
  onInsertHeading,
  onInsertList,
}: ZoneToolbarProps) {
  return (
    <div data-testid={`zone-toolbar-${zoneId}`} role="toolbar">
      {contentPolicy.allowTables && (
        <button type="button" data-testid="zone-btn-table" onClick={onInsertTable}>
          Table
        </button>
      )}
      {contentPolicy.allowImages && (
        <button type="button" data-testid="zone-btn-image" onClick={onInsertImage}>
          Image
        </button>
      )}
      {contentPolicy.allowHeadings && (
        <button type="button" data-testid="zone-btn-heading" onClick={onInsertHeading}>
          Heading
        </button>
      )}
      {contentPolicy.allowLists && (
        <button type="button" data-testid="zone-btn-list" onClick={onInsertList}>
          List
        </button>
      )}
    </div>
  );
}
