import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getActiveAlerts, acknowledgeAlert } from '@/api/alerts';
import { notification } from 'antd';
import type { Alert } from '@/types/alert';

export const useActiveAlerts = () =>
  useQuery({
    queryKey: ['alerts', 'active'],
    queryFn: async () => {
      const res = await getActiveAlerts();
      const payload = res.data as unknown as { source: string; data: Alert[] };
      return payload.data ?? [];
    },
    refetchInterval: 30_000,
    staleTime: 10_000,
  });

export const useAcknowledgeAlert = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (alertId: number) => acknowledgeAlert(alertId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alerts', 'active'] });
      notification.success({ message: 'Alert acknowledged' });
    },
  });
};
