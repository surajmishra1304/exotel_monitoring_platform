import React from 'react';
import { Spin } from 'antd';

interface Props {
  size?: 'small' | 'default' | 'large';
}

const LoadingSpinner: React.FC<Props> = ({ size = 'default' }) => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 48 }}>
    <Spin size={size} />
  </div>
);

export default LoadingSpinner;
