import { useQuery } from '@tanstack/react-query';
import { getDashboardSummary } from '@/api/dashboard';

export const useDashboardSummary = () =>
  useQuery({
    queryKey: ['dashboard', 'summary'],
    queryFn: async () => {
      const res = await getDashboardSummary();
      const p = res.data;
      return { source: p.source, data: p.data ?? [] };
    },
    refetchInterval: 60_000,
    staleTime: 30_000,
  });
