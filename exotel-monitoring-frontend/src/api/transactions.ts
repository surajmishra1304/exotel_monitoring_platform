import client from './client';
import { JobTransaction } from '@/types/transaction';

export interface TransactionFilters {
  exophone_id?: number;
  status?: string;
  from?: string;
  to?: string;
  limit?: number;
}

export interface TransactionsResponse {
  data: JobTransaction[];
  total: number;
  limit: number;
}

export const getTransactions = (filters: TransactionFilters) =>
  client.get<TransactionsResponse>('/api/v1/transactions', { params: filters });
