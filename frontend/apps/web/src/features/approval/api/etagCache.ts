const cache = new Map<string, string>();

export const etagCache = {
  get: (resourceId: string): string | undefined => cache.get(resourceId),
  set: (resourceId: string, etag: string): void => {
    cache.set(resourceId, etag);
  },
  delete: (resourceId: string): void => {
    cache.delete(resourceId);
  },
  clear: (): void => {
    cache.clear();
  },
};
