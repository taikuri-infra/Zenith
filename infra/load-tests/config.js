// Shared configuration for Zenith k6 load tests.
// Override via environment variables:
//   k6 run -e API_URL=https://api.example.com -e SMOKE_TEST_EMAIL=... script.js

export const BASE_URL = __ENV.API_URL || 'https://api.stage.freezenith.com';
export const EMAIL = __ENV.SMOKE_TEST_EMAIL || 'smoke-ci@zenith.dev';
export const PASSWORD = __ENV.SMOKE_TEST_PASSWORD || 'SmokeTest1234';

// APISIX rate limit: 100 req/60s per IP.
// Helper to calculate safe sleep between requests.
// With N VUs, each VU should sleep at least (N * 60 / 100) seconds
// to stay under the global rate limit. We add a buffer.
export function rateLimitSleep(vus) {
  const minSleep = (vus * 60) / 100;
  // Add 20% buffer to stay safely under the limit
  return Math.max(minSleep * 1.2, 1);
}
