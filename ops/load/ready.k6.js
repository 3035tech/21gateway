import http from 'k6/http'
import { check } from 'k6'

// Readiness only: no credentials, SMS, callbacks or state-consuming queries.
const base = __ENV.BASE_URL || 'http://127.0.0.1:8080'
if (!/^http:\/\/(127\.0\.0\.1|localhost):[0-9]{1,5}$/.test(base)) {
  throw new Error('Use a loopback HTTP URL with an explicit port.')
}
export const options = {
  scenarios: {
    readiness: {
      executor: 'constant-arrival-rate',
      rate: 50,
      timeUnit: '1s',
      duration: '30s',
      preAllocatedVUs: 10,
      maxVUs: 50,
    },
  },
  thresholds: { http_req_failed: ['rate<0.01'], checks: ['rate>0.99'] },
}
export default function () {
  const response = http.get(`${base}/readyz`, { timeout: '5s', redirects: 0 })
  check(response, { 'database ready': (r) => r.status === 200 })
}
