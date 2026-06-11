import React from 'react';
import { Empty, Button } from 'antd';

interface Props {
  description?: string;
  onRetry?: () => void;
}

const EmptyState: React.FC<Props> = ({ description = 'No data available', onRetry }) => (
  <div style={{ padding: 48, textAlign: 'center' }}>
    <Empty description={description} />
    {onRetry && (
      <Button onClick={onRetry} style={{ marginTop: 16 }}>
        Retry
      </Button>
    )}
  </div>
);

export default EmptyState;
