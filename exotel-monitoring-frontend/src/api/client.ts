import axios, { AxiosError } from 'axios';
import { notification } from 'antd';

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
});

client.interceptors.request.use((config) => {
  const user = localStorage.getItem('ops_user') ?? 'OPS_CONSOLE';
  config.headers['X-User'] = user;
  return config;
});

client.interceptors.response.use(
  (res) => res,
  (err: AxiosError<{ error?: string; mysql?: string; redis?: string; mysql_error?: string; redis_error?: string }>) => {
    const status = err.response?.status;
    const data = err.response?.data;
    const url = err.config?.url ?? 'unknown endpoint';

    let title = 'Request Failed';
    let description = '';

    if (!err.response) {
      // Network-level error — backend not reachable at all.
      title = 'Cannot reach server';
      description = 'The monitoring service is not responding. Verify the backend is running on port 8080.';
    } else if (status === 503 && data?.mysql && data?.redis) {
      // Health-check 503 should be handled by the dedicated healthClient.
      // If it somehow lands here, surface the specific service failures.
      title = 'Service Degraded';
      const issues: string[] = [];
      if (data.mysql !== 'CONNECTED') issues.push(`MySQL: ${data.mysql_error ?? 'disconnected'}`);
      if (data.redis !== 'CONNECTED') issues.push(`Redis: ${data.redis_error ?? 'disconnected'}`);
      description = issues.join(' · ') || 'One or more dependencies are down.';
    } else if (status === 500) {
      title = 'Server Error';
      description = data?.error ?? `Internal server error on ${url}`;
    } else if (status === 404) {
      title = 'Not Found';
      description = data?.error ?? `Resource not found at ${url}`;
    } else if (status === 400) {
      title = 'Bad Request';
      description = data?.error ?? `Invalid request to ${url}`;
    } else if (status === 503) {
      title = 'Service Unavailable';
      description = data?.error ?? 'A backend dependency (MySQL or Redis) is currently unreachable.';
    } else {
      title = `API Error ${status ?? ''}`.trim();
      description = data?.error ?? err.message;
    }

    notification.error({ message: title, description, duration: 6 });
    const error = new Error(description) as Error & { status?: number };
    error.status = status;
    return Promise.reject(error);
  }
);

export default client;
