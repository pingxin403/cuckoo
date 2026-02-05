/**
 * Workload Configuration
 * 
 * Defines different load patterns for testing.
 * Usage: k6 run -e WORKLOAD=stress test.js
 */

export const WorkloadConfig = {
  // Smoke test - minimal load for quick validation
  smoke: {
    executor: 'constant-vus',
    vus: 5,
    duration: '1m',
  },
  
  // Load test - normal expected load
  load: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '2m', target: 100 },   // Ramp up
      { duration: '10m', target: 100 },  // Steady state
      { duration: '2m', target: 0 },     // Ramp down
    ],
    gracefulRampDown: '30s',
  },
  
  // Stress test - find breaking point
  stress: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '2m', target: 100 },
      { duration: '5m', target: 200 },
      { duration: '5m', target: 300 },
      { duration: '5m', target: 400 },
      { duration: '2m', target: 0 },
    ],
    gracefulRampDown: '30s',
  },
  
  // Spike test - sudden traffic surge
  spike: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '10s', target: 100 },
      { duration: '1m', target: 1000 },  // Spike
      { duration: '3m', target: 1000 },  // Hold
      { duration: '10s', target: 100 },
      { duration: '3m', target: 100 },   // Recovery
    ],
    gracefulRampDown: '30s',
  },
  
  // Sustained high load - constant arrival rate
  sustained: {
    executor: 'constant-arrival-rate',
    rate: 100000,
    timeUnit: '1s',
    duration: '10m',
    preAllocatedVUs: 1000,
    maxVUs: 2000,
  },
};

/**
 * Get workload configuration
 * @returns {Object} Workload configuration
 */
export function getWorkload() {
  const workload = __ENV.WORKLOAD || 'smoke';
  const config = WorkloadConfig[workload];
  
  if (!config) {
    throw new Error(`Unknown workload: ${workload}. Available: ${Object.keys(WorkloadConfig).join(', ')}`);
  }
  
  console.log(`Using workload: ${workload}`);
  return config;
}
