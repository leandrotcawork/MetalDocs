import { useCallback, useState } from 'react';
import { getPlaceholderValues } from '../api/documentsV2';
import type { PlaceholderValueDTO } from '../api/documentsV2';

export function useComputedRefresh(
  docId: string,
  onRefreshed: (values: PlaceholderValueDTO[]) => void,
): { triggerRefresh: () => void; refreshing: boolean } {
  const [refreshing, setRefreshing] = useState(false);

  const triggerRefresh = useCallback(() => {
    setRefreshing(true);
    getPlaceholderValues(docId)
      .then((values) => {
        onRefreshed(values);
      })
      .finally(() => {
        setRefreshing(false);
      });
  }, [docId, onRefreshed]);

  return { triggerRefresh, refreshing };
}
