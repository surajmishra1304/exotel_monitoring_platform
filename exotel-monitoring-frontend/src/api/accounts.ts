import client from './client';
import type { Exophone } from '@/types/exophone';

export interface CreateExophoneInput {
  number: string;
  priority: 'P0' | 'P1' | 'P2' | 'P3';
}

export interface CreateAccountPayload {
  account_name: string;
  sid: string;
  api_key: string;
  api_token: string;
  subdomain: string;
  cluster: string;
  exophones: CreateExophoneInput[];
}

export interface CreateAccountResponse {
  account_id: number;
  account_name: string;
  exophones_created: number;
  jobs_created: number;
  message: string;
}

export const createAccount = (payload: CreateAccountPayload) =>
  client.post<CreateAccountResponse>('/api/v1/accounts', payload);

export const getExophone = (id: number) =>
  client.get<{ data: Exophone }>(`/api/v1/exophones/${id}`);
