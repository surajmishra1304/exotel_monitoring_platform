import client from './client';
import { Alert } from '@/types/alert';

export interface CreateAlertPayload {
  transaction_id: string;
  exophone_id: number;
  account_id: number;
  alert_type: string;
  severity: string;
  message: string;
  channel: string;
}

export const getActiveAlerts = () =>
  client.get<{ data: Alert[] }>('/api/v1/alerts/active');

export const acknowledgeAlert = (alertId: number) =>
  client.put<{ data: Alert }>(`/api/v1/alerts/${alertId}/acknowledge`);

export const createAlert = (payload: CreateAlertPayload) =>
  client.post<{ data: Alert }>('/api/v1/alerts', payload);
