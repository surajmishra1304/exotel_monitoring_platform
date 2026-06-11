import React from 'react';
import { Badge } from 'antd';
import type { BadgeProps } from 'antd';
import { HEARTBEAT_COLOR, HEARTBEAT_LABEL } from '@/utils/constants';
import type { HeartbeatStatus } from '@/types/snapshot';

const STATUS_MAP: Record<HeartbeatStatus, BadgeProps['status']> = {
  OK: 'success',
  DEGRADED: 'warning',
  OUTAGE: 'error',
  UNKNOWN: 'default',
};

interface Props {
  status: HeartbeatStatus;
  showLabel?: boolean;
}

const HeartbeatBadge: React.FC<Props> = ({ status, showLabel = true }) => (
  <Badge
    status={STATUS_MAP[status] ?? 'default'}
    color={STATUS_MAP[status] ? undefined : HEARTBEAT_COLOR[status]}
    text={showLabel ? HEARTBEAT_LABEL[status] : undefined}
  />
);

export default HeartbeatBadge;
