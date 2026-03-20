import http from "k6/http";
import { check, sleep } from "k6";

// Technical perf mode only. For these scripts, enable METALDOCS_AUTH_LEGACY_HEADER_ENABLED=true
// or migrate the harness to a cookie-session login flow before using them in official runtime validation.
const BASE_URL = __ENV.BASE_URL || "http://localhost:8080/api/v1";
const USER_ID = __ENV.USER_ID || "admin-local";
const PROFILE_CODE = __ENV.PROFILE_CODE || "po";
const DOCUMENT_ID = __ENV.DOCUMENT_ID || "";
const PDF_AVAILABLE = __ENV.PDF_AVAILABLE === "true";

export const options = {
  scenarios: {
    light_concurrency: {
      executor: "ramping-arrival-rate",
      startRate: 5,
      timeUnit: "1s",
      preAllocatedVUs: 20,
      maxVUs: 60,
      stages: [
        { target: 10, duration: "1m" },
        { target: 20, duration: "2m" },
        { target: 0, duration: "1m" },
      ],
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<1200", "p(99)<2000"],
  },
};

function authHeaders() {
  return { "X-User-Id": USER_ID };
}

export default function () {
  const profiles = http.get(`${BASE_URL}/document-profiles`, {
    headers: authHeaders(),
    tags: { operation: "list_profiles" },
  });
  check(profiles, {
    "profiles status 200": (r) => r.status === 200,
  });

  const schemas = http.get(
    `${BASE_URL}/document-profiles/${PROFILE_CODE}/schema`,
    {
      headers: authHeaders(),
      tags: { operation: "list_profile_schemas" },
    }
  );
  check(schemas, {
    "schema status 200": (r) => r.status === 200,
  });

  const processAreas = http.get(`${BASE_URL}/process-areas`, {
    headers: authHeaders(),
    tags: { operation: "list_process_areas" },
  });
  check(processAreas, {
    "process areas status 200": (r) => r.status === 200,
  });

  const subjects = http.get(`${BASE_URL}/document-subjects`, {
    headers: authHeaders(),
    tags: { operation: "list_subjects" },
  });
  check(subjects, {
    "subjects status 200": (r) => r.status === 200,
  });

  if (DOCUMENT_ID) {
    const nativeContent = http.get(
      `${BASE_URL}/documents/${DOCUMENT_ID}/content/native`,
      {
        headers: authHeaders(),
        tags: { operation: "get_content_native" },
      }
    );
    check(nativeContent, {
      "native content status 200": (r) => r.status === 200,
    });

    if (PDF_AVAILABLE) {
      const pdf = http.get(`${BASE_URL}/documents/${DOCUMENT_ID}/content/pdf`, {
        headers: authHeaders(),
        tags: { operation: "get_content_pdf" },
      });
      check(pdf, {
        "pdf status 200": (r) => r.status === 200,
      });
    }
  }

  sleep(0.2);
}
