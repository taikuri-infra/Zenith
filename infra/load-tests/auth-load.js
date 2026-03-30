import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { BASE_URL, EMAIL, PASSWORD, rateLimitSleep } from './config.js';

// Custom metrics
const loginDuration = new Trend('login_duration', true);
const loginFailRate = new Rate('login_failures');
const meDuration = new Trend('me_endpoint_duration', true);

export const options = {
  scenarios: {
    login_flow: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 10 },  // ramp up to 10 VUs
        { duration: '1m', target: 10 },   // hold at 10 VUs
        { duration: '30s', target: 0 },   // ramp down
      ],
      exec: 'loginFlow',
      gracefulRampDown: '10s',
    },
    token_refresh: {
      executor: 'constant-vus',
      vus: 5,
      duration: '2m',
      exec: 'tokenRefresh',
      startTime: '10s', // slight offset to avoid initial burst
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.05'],
    login_duration: ['p(95)<2000'],
    login_failures: ['rate<0.05'],
    me_endpoint_duration: ['p(95)<1500'],
  },
};

// Login helper — returns token response or null on failure.
function doLogin() {
  const res = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: EMAIL, password: PASSWORD }),
    {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'POST /api/v1/auth/login' },
    }
  );

  loginDuration.add(res.timings.duration);

  const success = check(res, {
    'login status is 200': (r) => r.status === 200,
    'login returns access_token': (r) => {
      try {
        const body = r.json();
        return body.access_token !== undefined && body.access_token !== '';
      } catch (e) {
        return false;
      }
    },
    'login returns refresh_token': (r) => {
      try {
        return r.json().refresh_token !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  loginFailRate.add(!success);

  if (!success) {
    console.warn(`Login failed: status=${res.status} body=${res.body}`);
    return null;
  }

  return res.json();
}

// setup() runs once before all VUs. Returns shared data.
export function setup() {
  const tokenData = doLogin();
  if (!tokenData) {
    throw new Error('Setup login failed — cannot proceed with load test');
  }
  return {
    accessToken: tokenData.access_token,
    refreshToken: tokenData.refresh_token,
  };
}

// Scenario: repeated login flow (tests auth system under load)
export function loginFlow() {
  group('Login Flow', () => {
    doLogin();
  });

  // With 10 VUs and 100 req/60s limit, each VU needs ~6s between requests.
  // Login is a single request, so sleep accordingly.
  sleep(rateLimitSleep(10));
}

// Scenario: authenticated requests to /auth/me (token validation under load)
export function tokenRefresh(data) {
  group('Token Refresh / Me', () => {
    const res = http.get(`${BASE_URL}/api/v1/auth/me`, {
      headers: {
        Authorization: `Bearer ${data.accessToken}`,
        'Content-Type': 'application/json',
      },
      tags: { name: 'GET /api/v1/auth/me' },
    });

    meDuration.add(res.timings.duration);

    const ok = check(res, {
      'me status is 200': (r) => r.status === 200,
      'me returns user data': (r) => {
        try {
          const body = r.json();
          return body.email !== undefined || body.id !== undefined;
        } catch (e) {
          return false;
        }
      },
    });

    if (!ok && res.status === 429) {
      console.warn('Rate limited on /auth/me — backing off');
      sleep(5);
    }
  });

  // With 5 VUs: each VU needs ~3.6s between requests
  sleep(rateLimitSleep(5));
}
