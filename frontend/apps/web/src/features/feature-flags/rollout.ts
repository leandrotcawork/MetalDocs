// Deterministic per-user canary gate. Bucket is derived from FNV-1a over
// the user ID, giving a stable 0-99 value independent of process restarts.

function fnv1a(input: string): number {
  let hash = 0x811c9dc5;
  for (let i = 0; i < input.length; i++) {
    hash ^= input.charCodeAt(i);
    hash = (hash * 0x01000193) >>> 0;
  }
  return hash;
}

export function rolloutBucketForUser(userId: string): number {
  if (!userId) return -1;
  return fnv1a(`mddm-rollout:${userId}`) % 100;
}

export function isInRolloutBucket(userId: string, percent: number): boolean {
  if (!userId) return false;
  if (percent <= 0) return false;
  if (percent >= 100) return true;
  return rolloutBucketForUser(userId) < percent;
}
