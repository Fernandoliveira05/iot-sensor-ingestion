import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    telemetria: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      preAllocatedVUs: 20,
      maxVUs: 300,
      stages: [
        { target: 150, duration: '30s' },
        { target: 300, duration: '30s' },
        { target: 600, duration: '30s' },
        { target: 0, duration: '10s' },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<1000'],
  },
};

export default function () {
  const payload = JSON.stringify({
    device_id: `${__VU}-${__ITER}`,
    sensor_type: 'temperature',
    reading_type: 'analog',
    value: Math.random() * 15 + 20,
    timestamp: new Date().toISOString(),
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post('http://backend:8080/telemetria', payload, params);

  check(res, {
    'status 202': (r) => r.status === 202,
  });

  sleep(1);
}