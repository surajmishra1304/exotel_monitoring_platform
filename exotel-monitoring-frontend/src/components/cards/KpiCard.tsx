import React from 'react';
import { Card, Statistic, Skeleton } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined } from '@ant-design/icons';

interface Props {
  title: string;
  value: string | number;
  unit?: string;
  trend?: 'up' | 'down' | 'neutral';
  highlight?: 'success' | 'warning' | 'danger';
  loading?: boolean;
}

const HIGHLIGHT_COLOR: Record<string, string> = {
  success: '#52c41a',
  warning: '#faad14',
  danger: '#ff4d4f',
};

const KpiCard: React.FC<Props> = ({ title, value, unit, trend, highlight, loading }) => {
  if (loading) {
    return (
      <Card>
        <Skeleton active paragraph={false} />
      </Card>
    );
  }

  const valueColor = highlight ? HIGHLIGHT_COLOR[highlight] : undefined;
  const suffix = unit ? <span style={{ fontSize: 14 }}>{unit}</span> : undefined;
  const prefix =
    trend === 'up' ? (
      <ArrowUpOutlined style={{ color: '#52c41a' }} />
    ) : trend === 'down' ? (
      <ArrowDownOutlined style={{ color: '#ff4d4f' }} />
    ) : undefined;

  return (
    <Card>
      <Statistic
        title={title}
        value={value}
        suffix={suffix}
        prefix={prefix}
        valueStyle={valueColor ? { color: valueColor } : undefined}
      />
    </Card>
  );
};

export default KpiCard;
