// Asset URL allowlist. Only the MetalDocs image endpoint is permitted,
// keyed by UUID. All external or alternate paths are rejected.

const IMAGE_PATH_REGEX = /^\/api\/images\/[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;

export function isAllowlistedAssetUrl(url: string): boolean {
  if (typeof url !== "string" || url.length === 0) {
    return false;
  }

  // Reject dangerous pseudo-protocols explicitly.
  const lowered = url.toLowerCase().trim();
  if (lowered.startsWith("javascript:") || lowered.startsWith("data:") || lowered.startsWith("file:")) {
    return false;
  }

  // Extract pathname: either a relative /api/images/... or an absolute URL.
  let pathname: string;
  if (url.startsWith("/")) {
    pathname = url;
  } else {
    try {
      const parsed = new URL(url, "https://placeholder.local");
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
        return false;
      }
      pathname = parsed.pathname;
    } catch {
      return false;
    }
  }

  return IMAGE_PATH_REGEX.test(pathname);
}
