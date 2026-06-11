import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createAccount } from '@/api/accounts';
import type { CreateAccountPayload } from '@/api/accounts';
import { notification } from 'antd';

export const useCreateAccount = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateAccountPayload) => createAccount(payload),
    onSuccess: (res) => {
      // Invalidate dashboard summary so new account appears immediately.
      qc.invalidateQueries({ queryKey: ['dashboard', 'summary'] });
      notification.success({
        message: 'Organisation created',
        description: res.data.message,
        duration: 6,
      });
    },
  });
};
