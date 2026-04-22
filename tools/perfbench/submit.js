/**
 * k6 perf benchmark — submit flow
 * Threshold: p99 ≤ 200ms at 100 rps × 60s
 *
 * Usage:
 *   k6 run tools/perfbench/submit.js \
 *     -e BASE_URL=http://localhost:8080 \
 *     -e TENANT_ID=perf_tenant \
 *     -e SESSION_COOKIE=<cookie>
 */
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const thresholds = JSON.parse(open('./thresholds.json'));

export const options = {
  scenarios: {
    submit_rps: {
      executor: 'constant-arrival-rate',
      rate: __ENV.REDUCED === '1' ? 10 : 100,
      timeUnit: '1s',
      duration: __ENV.REDUCED === '1' ? '30s' : '60s',
      preAllocatedVUs: 20,
      maxVUs: 50,
    },
  },
  thresholds: {
    'http_req_duration{scenario:submit_rps}': [
      `p(99)<${__ENV.REDUCED === '1' ? thresholds.submit_p99_ms * 2 : thresholds.submit_p99_ms}`,
    ],
    'http_req_failed{scenario:submit_rps}': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TENANT_ID = __ENV.TENANT_ID || 'perf_tenant';
const SESSION = __ENV.SESSION_COOKIE || '';

let docCounter = 0;

export function setup() {
  // Create a pool of doc IDs to submit
  return { docs: Array.from({ length: 500 }, () => uuidv4()) };
}

export default function (data) {
  const docId = data.docs[docCounter++ % data.docs.length];
  const idempotencyKey = uuidv4();

  const res = http.post(
    `${BASE_URL}/api/v2/documents/${docId}/submit`,
    JSON.stringify({ routeId: __ENV.ROUTE_ID }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': idempotencyKey,
        'X-Tenant-ID': TENANT_ID,
        Cookie: `metaldocs_session=${SESSION}`,
      },
      tags: { scenario: 'submit_rps' },
    }
  );

  check(res, {
    'status 2xx or 409 (already submitted)': r =>
      r.status >= 200 && r.status < 300 || r.status === 409,
  });
}
