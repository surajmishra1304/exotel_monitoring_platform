import { useQuery } from '@tanstack/react-query';
import { getExophone } from '@/api/accounts';
import type { Exophone } from '@/types/exophone';

export const useExophoneById = (id: number) =>
  useQuery({
    queryKey: ['exophone', id],
    queryFn: async () => {
      const res = await getExophone(id);
      const payload = res.data as unknown as { data: Exophone };
      return payload.data;
    },
    enabled: id > 0,
    staleTime: 60_000,
  });
