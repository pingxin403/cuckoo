/**
 * Environment Configuration
 * 
 * Defines configuration for different deployment environments.
 * Usage: k6 run -e ENVIRONMENT=staging test.js
 */

export const EnvironmentConfig = {
  local: {
    baseUrl: 'http://localhost:8080',
    grpcUrl: 'localhost:50051',
  },
  dev: {
    baseUrl: 'https://dev-shortener.example.com',
    grpcUrl: 'dev-shortener.example.com:50051',
  },
  staging: {
    baseUrl: 'https://staging-shortener.example.com',
    grpcUrl: 'staging-shortener.example.com:50051',
  },
  prod: {
    baseUrl: 'https://api.example.com',
    grpcUrl: 'api.example.com:50051',
  },
};

/**
 * Get environment configuration
 * @returns {Object} Environment configuration
 */
export function getEnvironment() {
  const env = __ENV.ENVIRONMENT || 'local';
  const config = EnvironmentConfig[env];
  
  if (!config) {
    throw new Error(`Unknown environment: ${env}. Available: ${Object.keys(EnvironmentConfig).join(', ')}`);
  }
  
  console.log(`Using environment: ${env}`);
  return config;
}
