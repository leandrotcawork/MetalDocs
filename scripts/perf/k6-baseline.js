import http from "k6/http";
import { check, sleep } from "k6";

// Technical perf mode only. For these scripts, enable METALDOCS_AUTH_LEGACY_HEADER_ENABLED=true
// or migrate the harness to a cookie-session login flow before using them in official runtime validation.
const BASE_URL = __ENV.BASE_URL || "http://localhost:8080/api/v1";
const USER_ID = __ENV.USER_ID || "admin-local";

export const options = {
  scenarios: {
    read_heavy: {
      executor: "ramping-arrival-rate",
      startRate: 5,
      timeUnit: "1s",
      preAllocatedVUs: 20,
      maxVUs: 80,
      stages: [
        { target: 20, duration: "1m" },
        { target: 50, duration: "3m" },
        { target: 0, duration: "1m" },
      ],
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<1200", "p(99)<2000"],
  },
};

export default function () {
  const search = http.get(`${BASE_URL}/search/documents?limit=10`, {
    headers: { "X-User-Id": USER_ID },
  });
  check(search, {
    "search status 200": (r) => r.status === 200,
  });

  const health = http.get(`${BASE_URL}/health/live`);
  check(health, {
    "health status 200": (r) => r.status === 200,
  });

  sleep(0.2);
}
