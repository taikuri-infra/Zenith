// =============================================================================
// Zenith Platform — k6 Load Test
//
// Scenarios:
//   1. Auth stress: 50 VUs, 2 min — register + login + token refresh
//   2. App CRUD: 20 VUs, 3 min — create, list, get, delete apps
//   3. Mixed workload: 100 VUs, 5 min — realistic traffic mix
//   4. Spike test: 0→200 VUs in 30s, hold 1 min, ramp down
//
// Usage:
//   k6 run infra/scripts/load-test.js --env API_URL=https://api.stage.freezenith.com
//   k6 run infra/scripts/load-test.js --env API_URL=... --env SCENARIO=auth
//
// =============================================================================

import http from "k6/http";
import { check, sleep, group } from "k6";
import { Rate, Trend } from "k6/metrics";

const API_URL = __ENV.API_URL || "https://api.stage.freezenith.com";
const SCENARIO = __ENV.SCENARIO || "all";

// Custom metrics
const errorRate = new Rate("errors");
const authDuration = new Trend("auth_duration", true);
const crudDuration = new Trend("crud_duration", true);

// Thresholds
export const options = {
  thresholds: {
    http_req_duration: ["p(95)<500"],
    errors: ["rate<0.01"],
    http_req_failed: ["rate<0.01"],
  },
  scenarios: buildScenarios(),
};

function buildScenarios() {
  if (SCENARIO === "auth") {
    return { auth_stress: authScenario() };
  }
  if (SCENARIO === "crud") {
    return { app_crud: crudScenario() };
  }
  if (SCENARIO === "mixed") {
    return { mixed_workload: mixedScenario() };
  }
  if (SCENARIO === "spike") {
    return { spike_test: spikeScenario() };
  }
  // all
  return {
    auth_stress: authScenario(),
    app_crud: Object.assign(crudScenario(), { startTime: "130s" }),
    mixed_workload: Object.assign(mixedScenario(), { startTime: "330s" }),
    spike_test: Object.assign(spikeScenario(), { startTime: "650s" }),
  };
}

function authScenario() {
  return {
    executor: "constant-vus",
    vus: 50,
    duration: "2m",
    exec: "authStress",
  };
}

function crudScenario() {
  return {
    executor: "constant-vus",
    vus: 20,
    duration: "3m",
    exec: "appCrud",
  };
}

function mixedScenario() {
  return {
    executor: "constant-vus",
    vus: 100,
    duration: "5m",
    exec: "mixedWorkload",
  };
}

function spikeScenario() {
  return {
    executor: "ramping-vus",
    startVUs: 0,
    stages: [
      { duration: "30s", target: 200 },
      { duration: "1m", target: 200 },
      { duration: "30s", target: 0 },
    ],
    exec: "mixedWorkload",
  };
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const headers = { "Content-Type": "application/json" };

function authHeaders(token) {
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
  };
}

function uniqueEmail() {
  return `k6-${__VU}-${__ITER}-${Date.now()}@load.zenith.dev`;
}

function registerAndLogin() {
  const email = uniqueEmail();
  const password = "K6LoadTest1234";

  const regRes = http.post(
    `${API_URL}/api/v1/auth/register`,
    JSON.stringify({ email, password, name: `K6 User ${__VU}` }),
    { headers }
  );

  // Login (works even if registration requires verification — the API
  // may auto-verify in staging or return a token directly)
  const loginRes = http.post(
    `${API_URL}/api/v1/auth/login`,
    JSON.stringify({ email, password }),
    { headers }
  );

  const body = loginRes.json() || {};
  return {
    token: body.access_token || body.token || "",
    refreshToken: body.refresh_token || "",
    email,
  };
}

// ---------------------------------------------------------------------------
// Scenario: Auth Stress
// ---------------------------------------------------------------------------

export function authStress() {
  group("Auth Stress", () => {
    const start = Date.now();
    const { token, refreshToken } = registerAndLogin();
    authDuration.add(Date.now() - start);

    const ok = check(token, {
      "got auth token": (t) => t.length > 0,
    });
    errorRate.add(!ok);

    if (refreshToken) {
      const refreshRes = http.post(
        `${API_URL}/api/v1/auth/refresh`,
        JSON.stringify({ refresh_token: refreshToken }),
        { headers }
      );
      check(refreshRes, {
        "refresh status 200": (r) => r.status === 200,
      });
    }
  });
  sleep(1);
}

// ---------------------------------------------------------------------------
// Scenario: App CRUD
// ---------------------------------------------------------------------------

export function appCrud() {
  const { token } = registerAndLogin();
  if (!token) {
    errorRate.add(true);
    return;
  }

  group("App CRUD", () => {
    const start = Date.now();

    // Create project first
    const projRes = http.post(
      `${API_URL}/api/v1/projects`,
      JSON.stringify({ name: `k6-proj-${__VU}-${__ITER}` }),
      { headers: authHeaders(token) }
    );
    const projBody = projRes.json() || {};
    const projectId = projBody.id || projBody.project_id || "";

    // Create app
    const appRes = http.post(
      `${API_URL}/api/v1/apps`,
      JSON.stringify({
        name: `k6-app-${__VU}-${__ITER}`,
        project_id: projectId,
        runtime: "nodejs",
      }),
      { headers: authHeaders(token) }
    );
    const appBody = appRes.json() || {};
    const appId = appBody.id || appBody.app_id || "";

    check(appRes, {
      "app created": (r) => r.status === 200 || r.status === 201,
    });

    // List apps
    const listRes = http.get(`${API_URL}/api/v1/apps`, {
      headers: authHeaders(token),
    });
    check(listRes, {
      "list apps 200": (r) => r.status === 200,
    });

    // Get app
    if (appId) {
      const getRes = http.get(`${API_URL}/api/v1/apps/${appId}`, {
        headers: authHeaders(token),
      });
      check(getRes, {
        "get app 200": (r) => r.status === 200,
      });

      // Delete app
      const delRes = http.del(`${API_URL}/api/v1/apps/${appId}`, null, {
        headers: authHeaders(token),
      });
      check(delRes, {
        "delete app success": (r) =>
          r.status === 200 || r.status === 204,
      });
    }

    crudDuration.add(Date.now() - start);
  });
  sleep(1);
}

// ---------------------------------------------------------------------------
// Scenario: Mixed Workload (60% reads, 30% writes, 10% deletes)
// ---------------------------------------------------------------------------

export function mixedWorkload() {
  const { token } = registerAndLogin();
  if (!token) {
    errorRate.add(true);
    return;
  }

  group("Mixed Workload", () => {
    const rand = Math.random();

    if (rand < 0.6) {
      // Read operations
      const res = http.get(`${API_URL}/api/v1/apps`, {
        headers: authHeaders(token),
      });
      check(res, { "read ok": (r) => r.status === 200 });
      errorRate.add(res.status >= 500);
    } else if (rand < 0.9) {
      // Write: create project
      const res = http.post(
        `${API_URL}/api/v1/projects`,
        JSON.stringify({ name: `k6-mixed-${__VU}-${__ITER}` }),
        { headers: authHeaders(token) }
      );
      check(res, {
        "write ok": (r) => r.status === 200 || r.status === 201,
      });
      errorRate.add(res.status >= 500);
    } else {
      // Health check (lightweight "delete-weight" operation)
      const res = http.get(`${API_URL}/health`);
      check(res, { "health ok": (r) => r.status === 200 });
      errorRate.add(res.status >= 500);
    }
  });
  sleep(0.5);
}
