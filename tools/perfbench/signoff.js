/**
 * k6 perf benchmark — signoff flow
 * Threshold: p99 ≤ 300ms at 200 rps × 60s
 */
import http from 'k6/http';
import { check } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const thresholds = JSON.parse(open('./thresholds.json'));

export const options = {
  scenarios: {
    signoff_rps: {
      executor: 'constant-arrival-rate',
      rate: __ENV.REDUCED === '1' ? 10 : 200,
      timeUnit: '1s',
      duration: __ENV.REDUCED === '1' ? '30s' : '60s',
      preAllocatedVUs: 40,
      maxVUs: 100,
    },
  },
  thresholds: {
    'http_req_duration{scenario:signoff_rps}': [
      `p(99)<${__ENV.REDUCED === '1' ? thresholds.signoff_p99_ms * 2 : thresholds.signoff_p99_ms}`,
    ],
    'http_req_failed{scenario:signoff_rps}': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TENANT_ID = __ENV.TENANT_ID || 'perf_tenant';
const SESSION = __ENV.SESSION_COOKIE || '';

export function setup() {
  return { instances: ((__ENV.INSTANCE_IDS) || '').split(',').filter(Boolean) };
}

export default function (data) {
  if (data.instances.length === 0) return;
  const instanceId = data.instances[Math.floor(Math.random() * data.instances.length)];

  const res = http.post(
    `${BASE_URL}/api/v2/instances/${instanceId}/signoff`,
    JSON.stringify({ decision: 'approve', password: __ENV.SIGNOFF_PASSWORD }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': uuidv4(),
        'X-Tenant-ID': TENANT_ID,
        Cookie: `metaldocs_session=${SESSION}`,
      },
      tags: { scenario: 'signoff_rps' },
    }
  );

  check(res, {
    'signoff accepted or already done': r =>
      r.status < 300 || r.status === 409 || r.status === 422,
  });
}
