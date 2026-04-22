/**
 * k6 perf benchmark — publish flow
 * Threshold: p99 ≤ 500ms at 50 rps × 60s
 */
import http from 'k6/http';
import { check } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const thresholds = JSON.parse(open('./thresholds.json'));

export const options = {
  scenarios: {
    publish_rps: {
      executor: 'constant-arrival-rate',
      rate: __ENV.REDUCED === '1' ? 10 : 50,
      timeUnit: '1s',
      duration: __ENV.REDUCED === '1' ? '30s' : '60s',
      preAllocatedVUs: 10,
      maxVUs: 30,
    },
  },
  thresholds: {
    'http_req_duration{scenario:publish_rps}': [
      `p(99)<${__ENV.REDUCED === '1' ? thresholds.publish_p99_ms * 2 : thresholds.publish_p99_ms}`,
    ],
    'http_req_failed{scenario:publish_rps}': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TENANT_ID = __ENV.TENANT_ID || 'perf_tenant';
const SESSION = __ENV.SESSION_COOKIE || '';

export function setup() {
  return { docs: ((__ENV.DOC_IDS) || '').split(',').filter(Boolean) };
}

export default function (data) {
  if (data.docs.length === 0) return;
  const docId = data.docs[Math.floor(Math.random() * data.docs.length)];

  // GET ETag first
  const getRes = http.get(`${BASE_URL}/api/v2/documents/${docId}`, {
    headers: { 'X-Tenant-ID': TENANT_ID, Cookie: `metaldocs_session=${SESSION}` },
  });
  const etag = getRes.headers['Etag'] || getRes.headers['ETag'] || '';

  const res = http.post(
    `${BASE_URL}/api/v2/documents/${docId}/publish`,
    JSON.stringify({}),
    {
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': uuidv4(),
        'If-Match': etag,
        'X-Tenant-ID': TENANT_ID,
        Cookie: `metaldocs_session=${SESSION}`,
      },
      tags: { scenario: 'publish_rps' },
    }
  );

  check(res, {
    'publish accepted or conflict': r => r.status < 300 || r.status === 409 || r.status === 412,
  });
}
