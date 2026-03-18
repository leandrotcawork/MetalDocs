import http from "k6/http";
import { check, sleep } from "k6";

// Technical perf mode only. For these scripts, enable METALDOCS_AUTH_LEGACY_HEADER_ENABLED=true
// or migrate the harness to a cookie-session login flow before using them in official runtime validation.
const BASE_URL = __ENV.BASE_URL || "http://localhost:8080/api/v1";
const USER_ID = __ENV.USER_ID || "admin-local";

export const options = {
  scenarios: {
    write_heavy: {
      executor: "ramping-arrival-rate",
      startRate: 2,
      timeUnit: "1s",
      preAllocatedVUs: 20,
      maxVUs: 100,
      stages: [
        { target: 10, duration: "1m" },
        { target: 25, duration: "2m" },
        { target: 0, duration: "1m" },
      ],
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<1500", "p(99)<2500"],
  },
};

function randomSuffix() {
  return `${Date.now()}-${__VU}-${__ITER}`;
}

export default function () {
  const suffix = randomSuffix();
  const createPayload = JSON.stringify({
    title: `Perf Document ${suffix}`,
    ownerId: `owner-${__VU}`,
    classification: "INTERNAL",
    initialContent: `content-${suffix}`,
  });

  const createRes = http.post(`${BASE_URL}/documents`, createPayload, {
    headers: {
      "Content-Type": "application/json",
      "X-User-Id": USER_ID,
    },
    tags: { operation: "create_document" },
  });

  const createOk = check(createRes, {
    "create status 201": (r) => r.status === 201,
    "create has documentId": (r) => {
      try {
        const body = JSON.parse(r.body);
        return !!body.documentId;
      } catch (_) {
        return false;
      }
    },
  });

  if (!createOk) {
    sleep(0.1);
    return;
  }

  const created = JSON.parse(createRes.body);
  const documentId = created.documentId;

  const transitionPayload = JSON.stringify({
    toStatus: "IN_REVIEW",
    reason: "k6 write concurrency test",
  });

  const transitionRes = http.post(
    `${BASE_URL}/workflow/documents/${documentId}/transitions`,
    transitionPayload,
    {
      headers: {
        "Content-Type": "application/json",
        "X-User-Id": USER_ID,
      },
      tags: { operation: "workflow_transition" },
    }
  );

  check(transitionRes, {
    "transition status 200": (r) => r.status === 200,
  });

  const versionsRes = http.get(
    `${BASE_URL}/documents/${documentId}/versions`,
    {
      headers: { "X-User-Id": USER_ID },
      tags: { operation: "list_versions" },
    }
  );

  check(versionsRes, {
    "versions status 200": (r) => r.status === 200,
    "versions has items": (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body.items) && body.items.length >= 1;
      } catch (_) {
        return false;
      }
    },
  });

  sleep(0.05);
}
