/**
 * k6 perf benchmark — scheduler tick latency
 * Threshold: ≤ 2s per 100 pending rows with 5 concurrent scheduler instances
 */
import http from 'k6/http';
import { check } from 'k6';

const thresholds = JSON.parse(open('./thresholds.json'));

export const options = {
  scenarios: {
    scheduler_tick: {
      executor: 'constant-vus',
      vus: 5,
      duration: '60s',
    },
  },
  thresholds: {
    'http_req_duration{scenario:scheduler_tick}': [
      `p(99)<${thresholds.scheduler_tick_ms}`,
    ],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const SESSION = __ENV.SESSION_COOKIE || '';

export default function () {
  // Trigger a scheduler tick via internal admin endpoint
  const res = http.post(
    `${BASE_URL}/internal/test/trigger-scheduler-tick`,
    null,
    {
      headers: { Cookie: `metaldocs_session=${SESSION}` },
      tags: { scenario: 'scheduler_tick' },
      timeout: '10s',
    }
  );

  check(res, {
    'tick completed': r => r.status === 200 || r.status === 204,
  });
}
