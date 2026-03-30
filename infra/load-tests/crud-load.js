import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { BASE_URL, EMAIL, PASSWORD, rateLimitSleep } from './config.js';

// Custom metrics
const listAppsDuration = new Trend('list_apps_duration', true);
const listProjectsDuration = new Trend('list_projects_duration', true);
const getPlanDuration = new Trend('get_plan_duration', true);
const listDatabasesDuration = new Trend('list_databases_duration', true);
const rateLimitHits = new Counter('rate_limit_hits');

export const options = {
  scenarios: {
    read_heavy: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 20 },  // ramp up
        { duration: '2m', target: 20 },   // sustain
        { duration: '30s', target: 0 },   // ramp down
      ],
      exec: 'readHeavy',
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<3000'],
    // Allow up to 10% failures (rate limiting is expected with 20 VUs)
    http_req_failed: ['rate<0.10'],
    list_apps_duration: ['p(95)<3000'],
    list_projects_duration: ['p(95)<3000'],
    get_plan_duration: ['p(95)<2000'],
    list_databases_duration: ['p(95)<3000'],
  },
};

// Login and create a test project for the load test.
export function setup() {
  // Login
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: EMAIL, password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  const loginOk = check(loginRes, {
    'setup: login succeeded': (r) => r.status === 200,
  });

  if (!loginOk) {
    throw new Error(`Setup login failed: status=${loginRes.status} body=${loginRes.body}`);
  }

  const token = loginRes.json().access_token;
  const authHeaders = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };

  // Create a test project for load testing
  const projectName = `load-test-${Date.now()}`;
  const createRes = http.post(
    `${BASE_URL}/api/v1/projects`,
    JSON.stringify({
      name: projectName,
      description: 'Temporary project for k6 load testing',
    }),
    { headers: authHeaders }
  );

  let projectId = null;
  if (createRes.status === 200 || createRes.status === 201) {
    try {
      const body = createRes.json();
      projectId = body.id || body.project_id;
      console.log(`Created test project: ${projectId} (${projectName})`);
    } catch (e) {
      console.warn(`Could not parse create project response: ${createRes.body}`);
    }
  } else {
    console.warn(`Could not create test project (status ${createRes.status}), proceeding without it`);
  }

  return {
    token: token,
    projectId: projectId,
    projectName: projectName,
  };
}

// Helper: make authenticated GET request with rate limit handling.
function authGet(url, token, tags) {
  const res = http.get(url, {
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    tags: tags,
  });

  if (res.status === 429) {
    rateLimitHits.add(1);
    console.warn(`Rate limited on ${tags.name} — backing off 5s`);
    sleep(5);
  }

  return res;
}

// Main scenario: cycle through read-only endpoints.
export function readHeavy(data) {
  const token = data.token;

  // 1. List apps
  group('List Apps', () => {
    const res = authGet(
      `${BASE_URL}/api/v1/apps`,
      token,
      { name: 'GET /api/v1/apps' }
    );
    listAppsDuration.add(res.timings.duration);

    check(res, {
      'list apps: status 200 or 429': (r) => r.status === 200 || r.status === 429,
    });
  });

  // Small sleep between requests within same iteration
  sleep(1);

  // 2. List projects
  group('List Projects', () => {
    const res = authGet(
      `${BASE_URL}/api/v1/projects`,
      token,
      { name: 'GET /api/v1/projects' }
    );
    listProjectsDuration.add(res.timings.duration);

    check(res, {
      'list projects: status 200 or 429': (r) => r.status === 200 || r.status === 429,
    });
  });

  sleep(1);

  // 3. Get plan
  group('Get Plan', () => {
    const res = authGet(
      `${BASE_URL}/api/v1/plan`,
      token,
      { name: 'GET /api/v1/plan' }
    );
    getPlanDuration.add(res.timings.duration);

    check(res, {
      'get plan: status 200 or 429': (r) => r.status === 200 || r.status === 429,
    });
  });

  sleep(1);

  // 4. List databases
  group('List Databases', () => {
    const res = authGet(
      `${BASE_URL}/api/v1/databases`,
      token,
      { name: 'GET /api/v1/databases' }
    );
    listDatabasesDuration.add(res.timings.duration);

    check(res, {
      'list databases: status 200 or 429': (r) => r.status === 200 || r.status === 429,
    });
  });

  // With 20 VUs doing 4 requests each per iteration, that is 80 requests per cycle.
  // At 100 req/60s limit, we need significant sleep between iterations.
  // Each VU should space out iterations by ~(20 * 60 / 100) * 1.2 = ~14.4s
  // But we already slept 3s (3x 1s sleeps), so sleep the remainder.
  sleep(rateLimitSleep(20) - 3);
}

// Teardown: delete the test project.
export function teardown(data) {
  if (!data.projectId) {
    console.log('No test project to clean up');
    return;
  }

  const res = http.del(
    `${BASE_URL}/api/v1/projects/${data.projectId}`,
    null,
    {
      headers: {
        Authorization: `Bearer ${data.token}`,
        'Content-Type': 'application/json',
      },
    }
  );

  if (res.status === 200 || res.status === 204) {
    console.log(`Cleaned up test project: ${data.projectId}`);
  } else {
    console.warn(`Failed to delete test project ${data.projectId}: status=${res.status}`);
  }
}
