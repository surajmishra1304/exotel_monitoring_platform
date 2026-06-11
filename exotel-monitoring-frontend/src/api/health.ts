import axios from 'axios';

// Dedicated client that never throws on 503 — lets the health page handle
// DEGRADED responses as data rather than errors.
const healthClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  timeout: 5000,
  validateStatus: () => true,
});

export interface HealthResponse {
  status: 'UP' | 'DEGRADED';
  mysql: 'CONNECTED' | 'DISCONNECTED';
  redis: 'CONNECTED' | 'DISCONNECTED';
  mysql_error?: string;
  redis_error?: string;
}

export const getSystemHealth = () =>
  healthClient.get<HealthResponse>('/health');
