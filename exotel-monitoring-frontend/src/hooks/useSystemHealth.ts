import { useQuery } from '@tanstack/react-query';
import { getSystemHealth } from '@/api/health';

export const useSystemHealth = () =>
  useQuery({
    queryKey: ['system', 'health'],
    queryFn: () => getSystemHealth().then((r) => r.data),
    refetchInterval: 15_000,
    staleTime: 10_000,
  });
