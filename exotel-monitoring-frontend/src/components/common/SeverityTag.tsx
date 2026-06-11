import React from 'react';
import { Tag } from 'antd';
import { SEVERITY_COLOR } from '@/utils/constants';
import type { Severity } from '@/types/alert';

interface Props {
  severity: Severity;
}

const SeverityTag: React.FC<Props> = ({ severity }) => (
  <Tag color={SEVERITY_COLOR[severity] ?? 'default'}>{severity}</Tag>
);

export default SeverityTag;
