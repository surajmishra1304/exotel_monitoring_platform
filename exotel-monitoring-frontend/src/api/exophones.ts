import client from './client';
import { Exophone } from '@/types/exophone';
import { ExophoneHealthSnapshot, CallMetricsSnapshot } from '@/types/snapshot';

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export const getExophones = (accountId: number, page = 1, limit = 20) =>
  client.get<PaginatedResponse<Exophone>>(`/api/v1/accounts/${accountId}/exophones`, {
    params: { page, limit },
  });

export const getExophoneHealthList = (accountId: number) =>
  client.get<{ data: ExophoneHealthSnapshot[] }>(`/api/v1/accounts/${accountId}/exophones/health`);

export const getExophoneMetrics = (id: number) =>
  client.get<{ data: ExophoneHealthSnapshot }>(`/api/v1/exophones/${id}/metrics`);

export const getTodayPerformance = (accountId: number, page = 1, limit = 20, date?: string) =>
  client.get<PaginatedResponse<CallMetricsSnapshot> & { date: string }>(
    `/api/v1/accounts/${accountId}/performance/today`,
    { params: { page, limit, ...(date ? { date } : {}) } },
  );

export const toggleCron = (exophoneId: number, isCronApplicable: 0 | 1) =>
  client.patch(`/api/v1/exophones/${exophoneId}/cron`, { is_cron_applicable: isCronApplicable });

export const updatePriority = (exophoneId: number, priority: string, frequencyMinutes?: number) =>
  client.patch(`/api/v1/exophones/${exophoneId}/priority`, {
    priority,
    ...(frequencyMinutes !== undefined ? { frequency_minutes: frequencyMinutes } : {}),
  });

export interface SyncExophonesResult {
  account_id: number;
  total_from_exotel: number;
  already_tracked: number;
  newly_added: number;
  added_numbers: string[];
}

export const syncExophones = (accountId: number) =>
  client.post<SyncExophonesResult>(`/api/v1/accounts/${accountId}/exophones/sync`);

export interface AddExophoneResult {
  exophone_id: number;
  exophone_number: string;
  account_id: number;
  priority: string;
  frequency_minutes: number;
  job_ids: number[];
  message: string;
}

export const addExophone = (accountId: number, number: string, priority: string) =>
  client.post<AddExophoneResult>(`/api/v1/accounts/${accountId}/exophones`, { number, priority });

export interface PriorityConfig {
  id: number;
  priority: string;
  frequency_minutes: number;
  updated_at: string;
  updated_by: string;
}

export const getPriorityConfigs = () =>
  client.get<{ data: PriorityConfig[] }>('/api/v1/priority-config');

export const updatePriorityConfig = (priority: string, frequencyMinutes: number) =>
  client.put(`/api/v1/priority-config/${priority}`, { frequency_minutes: frequencyMinutes });

export const toggleSkipCallLogs = (exophoneId: number, val: 0 | 1) =>
  client.patch<{ exophone_id: number; skip_call_logs: number; message: string }>(
    `/api/v1/exophones/${exophoneId}/skip-call-logs`,
    { skip_call_logs: val },
  );

export interface ReprocessSnapshotResult {
  source: string;
  exophone_id: number;
  exophone_number: string;
  date: string;
  unique_call_records: number;
  total_calls: number;
  connected_calls: number;
  answer_rate_pct: string;
  drop_rate_pct: string;
  leg1_drop_rate_pct: string;
}

export const reprocessSnapshot = (exophoneId: number, date: string) =>
  client.post<ReprocessSnapshotResult>(
    `/api/v1/exophones/${exophoneId}/snapshot/reprocess`,
    null,
    { params: { date } },
  );
