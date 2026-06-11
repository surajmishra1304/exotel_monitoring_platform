import React from 'react';
import { Tag } from 'antd';
import { TXN_STATUS_COLOR } from '@/utils/constants';
import type { TxnStatus } from '@/types/transaction';

interface Props {
  status: TxnStatus;
}

const StatusTag: React.FC<Props> = ({ status }) => (
  <Tag color={TXN_STATUS_COLOR[status] ?? 'default'}>{status}</Tag>
);

export default StatusTag;
