import { useQuery } from '@tanstack/react-query';
import { getExophoneMetrics } from '@/api/exophones';
import { getExophoneCallAnalytics } from '@/api/dashboard';
import type { ExophoneHealthSnapshot, CallMetricsSnapshot } from '@/types/snapshot';

export const useExophoneMetrics = (id: number) =>
  useQuery({
    queryKey: ['exophone', 'metrics', id],
    queryFn: async () => {
      const res = await getExophoneMetrics(id);
      const payload = res.data as unknown as { source: string; data: ExophoneHealthSnapshot };
      return payload.data;
    },
    enabled: id > 0,
    refetchInterval: 60_000,
    staleTime: 30_000,
  });

export interface CallAnalyticsMeta {
  data: CallMetricsSnapshot | null;
  stale: boolean;
  dataAsOf: string | null;
  pending: boolean;
  message: string | null;
  source: string;
}

export const useExophoneCallAnalytics = (id: number, date?: string) =>
  useQuery({
    queryKey: ['exophone', 'calls', id, date ?? 'today'],
    queryFn: async (): Promise<CallAnalyticsMeta> => {
      try {
        const res = await getExophoneCallAnalytics(id, date);
        const p = res.data;
        return {
          data:      p.data ?? null,
          stale:     p.stale ?? false,
          dataAsOf:  p.data_as_of ?? null,
          pending:   p.pending ?? false,
          message:   p.message ?? null,
          source:    p.source,
        };
      } catch (err: any) {
        // 404 with pending=true = no data yet, not an error
        const body = err?.response?.data ?? {};
        if (body.pending) {
          return { data: null, stale: false, dataAsOf: null, pending: true, message: body.error ?? null, source: 'none' };
        }
        throw err;
      }
    },
    enabled: id > 0,
    refetchInterval: (!date || date === 'today') ? 60_000 : false,
    staleTime:       (!date || date === 'today') ? 30_000  : Infinity,
  });
