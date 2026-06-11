import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getExophones, getExophoneHealthList, getTodayPerformance, toggleCron, updatePriority, syncExophones, addExophone, getPriorityConfigs, updatePriorityConfig, type PaginatedResponse, type SyncExophonesResult, type AddExophoneResult, type PriorityConfig } from '@/api/exophones';
import type { Exophone } from '@/types/exophone';
import type { ExophoneHealthSnapshot, CallMetricsSnapshot } from '@/types/snapshot';

export const useExophones = (accountId: number, page = 1, limit = 20) =>
  useQuery<PaginatedResponse<Exophone>>({
    queryKey: ['exophones', accountId, page, limit],
    queryFn: async () => {
      const res = await getExophones(accountId, page, limit);
      return res.data as unknown as PaginatedResponse<Exophone>;
    },
    enabled: accountId > 0,
    staleTime: 30_000,
    placeholderData: (prev) => prev,
  });

export const useExophoneHealthList = (accountId: number) =>
  useQuery({
    queryKey: ['exophones', 'health', accountId],
    queryFn: async () => {
      const res = await getExophoneHealthList(accountId);
      const payload = res.data as unknown as { data: ExophoneHealthSnapshot[] };
      return payload.data ?? [];
    },
    enabled: accountId > 0,
    refetchInterval: 30_000,
    staleTime: 15_000,
  });

export const useTodayPerformance = (accountId: number, page = 1, limit = 20, date?: string) =>
  useQuery<PaginatedResponse<CallMetricsSnapshot> & { date: string }>({
    queryKey: ['performance', accountId, page, limit, date ?? 'today'],
    queryFn: async () => {
      const res = await getTodayPerformance(accountId, page, limit, date);
      return res.data as unknown as PaginatedResponse<CallMetricsSnapshot> & { date: string };
    },
    enabled: accountId > 0,
    staleTime: 60_000,
    placeholderData: (prev) => prev,
  });

export const useToggleCron = (accountId: number) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ exophoneId, val }: { exophoneId: number; val: 0 | 1 }) =>
      toggleCron(exophoneId, val),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exophones', accountId] });
    },
  });
};

export const useUpdatePriority = (accountId: number) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ exophoneId, priority, frequencyMinutes }: {
      exophoneId: number;
      priority: string;
      frequencyMinutes?: number;
    }) => updatePriority(exophoneId, priority, frequencyMinutes),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exophones', accountId] });
    },
  });
};

export const useAddExophone = (accountId: number) => {
  const qc = useQueryClient();
  return useMutation<AddExophoneResult, Error, { number: string; priority: string }>({
    mutationFn: async ({ number, priority }) => {
      const res = await addExophone(accountId, number, priority);
      return res.data as unknown as AddExophoneResult;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exophones', accountId] });
    },
  });
};

export const usePriorityConfig = () => {
  const qc = useQueryClient();
  const query = useQuery<PriorityConfig[]>({
    queryKey: ['priority-config'],
    queryFn: async () => {
      const res = await getPriorityConfigs();
      const payload = res.data as unknown as { data: PriorityConfig[] };
      return payload.data ?? [];
    },
    staleTime: 60_000,
  });
  const mutation = useMutation({
    mutationFn: ({ priority, frequencyMinutes }: { priority: string; frequencyMinutes: number }) =>
      updatePriorityConfig(priority, frequencyMinutes),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['priority-config'] }),
  });
  return { ...query, updateConfig: mutation.mutate, updating: mutation.isPending };
};

export const useSyncExophones = (accountId: number) => {
  const qc = useQueryClient();
  return useMutation<SyncExophonesResult, Error>({
    mutationFn: async () => {
      const res = await syncExophones(accountId);
      return res.data as unknown as SyncExophonesResult;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exophones', accountId] });
    },
  });
};
