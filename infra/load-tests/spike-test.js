import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { BASE_URL, EMAIL, PASSWORD } from './config.js';

// Custom metrics
const healthDuration = new Trend('health_check_duration', true);
const appsDuration = new Trend('list_apps_spike_duration', true);
const rateLimitHits = new Counter('rate_limit_hits');
const errorRate = new Rate('error_rate');

export const options = {
  scenarios: {
    spike: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 50 },  // spike up to 50 VUs
        { duration: '1m', target: 50 },   // hold at 50
        { duration: '30s', target: 1 },   // ramp down
      ],
      gracefulRampDown: '15s',
    },
  },
  thresholds: {
    http_req_duration: ['p(99)<5000'],
    // Allow up to 20% errors — rate limiting is expected with 50 VUs
    http_req_failed: ['rate<0.20'],
    error_rate: ['rate<0.20'],
    health_check_duration: ['p(99)<3000'],
  },
};

// Setup: login once and return the token for all VUs.
export function setup() {
  const res = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: EMAIL, password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  const ok = check(res, {
    'setup: login succeeded': (r) => r.status === 200,
  });

  if (!ok) {
    throw new Error(`Setup login failed: status=${res.status} body=${res.body}`);
  }

  return {
    token: res.json().access_token,
  };
}

// Default function: hit health + list apps under spike conditions.
export default function (data) {
  // 1. Health check (unauthenticated — not routed through APISIX /api/* rate limit)
  group('Health Check', () => {
    const res = http.get(`${BASE_URL}/health`, {
      tags: { name: 'GET /health' },
    });

    healthDuration.add(res.timings.duration);

    const ok = check(res, {
      'health: status 200': (r) => r.status === 200,
    });

    errorRate.add(!ok);
  });

  sleep(0.5);

  // 2. List apps (authenticated — subject to APISIX rate limiting)
  group('List Apps (Spike)', () => {
    const res = http.get(`${BASE_URL}/api/v1/apps`, {
      headers: {
        Authorization: `Bearer ${data.token}`,
        'Content-Type': 'application/json',
      },
      tags: { name: 'GET /api/v1/apps (spike)' },
    });

    appsDuration.add(res.timings.duration);

    if (res.status === 429) {
      rateLimitHits.add(1);
      // 429 is expected under spike — don't count as error
      check(res, {
        'rate limited (expected under spike)': (r) => r.status === 429,
      });
    } else {
      const ok = check(res, {
        'list apps: status 200': (r) => r.status === 200,
      });
      errorRate.add(!ok);
    }
  });

  // Under spike conditions with 50 VUs, rate limiting is expected.
  // We intentionally push past the limit to test API resilience.
  // Sleep enough to get meaningful data without pure DoS.
  // 50 VUs * 2 requests per iteration = 100 req per cycle.
  // At 100 req/60s, we need ~60s per cycle, or ~30s per VU (since half are health).
  // But we want to actually spike, so use a shorter sleep to trigger rate limiting.
  sleep(2 + Math.random() * 3); // 2-5s random sleep to create realistic spike pattern
}
