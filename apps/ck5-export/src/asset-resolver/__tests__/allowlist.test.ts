import { describe, expect, it } from "vitest";
import { isAllowlistedAssetUrl } from "../allowlist";

describe("Asset URL allowlist", () => {
  it("allows /api/images/{uuid} URLs", () => {
    expect(isAllowlistedAssetUrl("/api/images/00000000-0000-4000-8000-000000000001")).toBe(true);
    expect(isAllowlistedAssetUrl("/api/images/abcdef12-3456-7890-abcd-ef1234567890")).toBe(true);
  });

  it("allows full URLs pointing at the same origin", () => {
    expect(isAllowlistedAssetUrl("https://metaldocs.example/api/images/00000000-0000-4000-8000-000000000001")).toBe(true);
  });

  it("rejects arbitrary external URLs", () => {
    expect(isAllowlistedAssetUrl("https://evil.example/pwn.png")).toBe(false);
    expect(isAllowlistedAssetUrl("http://attacker.net/image")).toBe(false);
  });

  it("rejects javascript: and data: protocols at the allowlist level", () => {
    expect(isAllowlistedAssetUrl("javascript:alert(1)")).toBe(false);
    expect(isAllowlistedAssetUrl("data:text/html,<script>")).toBe(false);
  });

  it("rejects non-UUID image paths", () => {
    expect(isAllowlistedAssetUrl("/api/images/../etc/passwd")).toBe(false);
    expect(isAllowlistedAssetUrl("/api/images/not-a-uuid")).toBe(false);
  });

  it("rejects paths outside /api/images/", () => {
    expect(isAllowlistedAssetUrl("/api/secrets/token")).toBe(false);
    expect(isAllowlistedAssetUrl("/api/images_v2/foo")).toBe(false);
  });
});
