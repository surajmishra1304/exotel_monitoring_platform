import { useQuery } from '@tanstack/react-query';
import { getDashboardSummary } from '@/api/dashboard';
import type { AccountDashboardSnapshot } from '@/types/snapshot';

export const useDashboardSummary = () =>
  useQuery({
    queryKey: ['dashboard', 'summary'],
    queryFn: async () => {
      const res = await getDashboardSummary();
      // Backend returns { source, data: AccountDashboardSnapshot[] }
      const payload = res.data as unknown as { source: string; data: AccountDashboardSnapshot[] };
      return { source: payload.source, data: payload.data ?? [] };
    },
    refetchInterval: 60_000,
    staleTime: 30_000,
  });
