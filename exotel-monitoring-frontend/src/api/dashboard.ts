import client from './client';
import type { AccountDashboardSnapshot, CallMetricsSnapshot } from '@/types/snapshot';

export interface DashboardSummaryResponse {
  data: AccountDashboardSnapshot[];
  source: string;
  generated_at: string;
}

export const getDashboardSummary = () =>
  client.get<DashboardSummaryResponse>('/api/v1/dashboard/summary');

export interface CallAnalyticsResponse {
  source: string;
  data: CallMetricsSnapshot;
  date: string;
  stale: boolean;
  data_as_of: string;
  message?: string;
  pending?: boolean;
}

export const getExophoneCallAnalytics = (exophoneId: number, date?: string) => {
  const params = date ? { date } : {};
  return client.get<CallAnalyticsResponse>(
    `/api/v1/dashboard/exophones/${exophoneId}/calls`,
    { params },
  );
};
