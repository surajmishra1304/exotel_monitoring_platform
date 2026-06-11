import React, { Component } from 'react';
import { Result, Button, Typography } from 'antd';

const { Text } = Typography;

interface State {
  hasError: boolean;
  error: Error | null;
}

interface Props {
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

// React class-based error boundary — catches render-time exceptions so
// a single broken component never takes down the whole page.
class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    // Log to console so developers see the full trace in DevTools.
    console.error('[ErrorBoundary] Uncaught render error:', error, info.componentStack);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <Result
          status="error"
          title="Something went wrong"
          subTitle={
            <Text type="secondary" style={{ wordBreak: 'break-word' }}>
              {this.state.error?.message ?? 'An unexpected render error occurred.'}
            </Text>
          }
          extra={
            <Button
              type="primary"
              onClick={() => this.setState({ hasError: false, error: null })}
            >
              Try Again
            </Button>
          }
        />
      );
    }
    return this.props.children;
  }
}

export default ErrorBoundary;
