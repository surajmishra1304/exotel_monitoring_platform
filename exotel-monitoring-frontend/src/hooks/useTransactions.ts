import { useQuery } from '@tanstack/react-query';
import { getTransactions } from '@/api/transactions';
import type { TransactionFilters } from '@/api/transactions';
import type { JobTransaction } from '@/types/transaction';

export const useTransactions = (filters: TransactionFilters) =>
  useQuery({
    queryKey: ['transactions', filters],
    queryFn: async () => {
      const res = await getTransactions(filters);
      const payload = res.data as unknown as { data: JobTransaction[]; count: number };
      return { data: payload.data ?? [], count: payload.count ?? 0 };
    },
    staleTime: 0,
  });
