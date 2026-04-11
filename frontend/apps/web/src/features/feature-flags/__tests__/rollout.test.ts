import { describe, expect, it } from "vitest";
import { isInRolloutBucket, rolloutBucketForUser } from "../rollout";

describe("rollout helper", () => {
  it("rolloutBucketForUser returns a stable integer in [0, 100) for a given user ID", () => {
    const a1 = rolloutBucketForUser("user-123");
    const a2 = rolloutBucketForUser("user-123");
    expect(a1).toBe(a2);
    expect(a1).toBeGreaterThanOrEqual(0);
    expect(a1).toBeLessThan(100);

    const b = rolloutBucketForUser("user-456");
    expect(b).not.toBe(a1);
  });

  it("isInRolloutBucket honors the percentage threshold", () => {
    expect(isInRolloutBucket("user-123", 0)).toBe(false);
    expect(isInRolloutBucket("user-123", 100)).toBe(true);
  });

  it("distributes users roughly uniformly across buckets", () => {
    let included = 0;
    for (let i = 0; i < 1000; i++) {
      if (isInRolloutBucket(`user-${i}`, 50)) included++;
    }
    expect(included).toBeGreaterThan(400);
    expect(included).toBeLessThan(600);
  });

  it("returns false for empty user ID (unauthenticated, never canary)", () => {
    expect(isInRolloutBucket("", 100)).toBe(false);
  });
});
