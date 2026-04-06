import * as React from "react";

type ToolUIErrorBoundaryProps = {
  children: React.ReactNode;
  fallback: React.ReactNode;
};

type ToolUIErrorBoundaryState = {
  hasError: boolean;
};

export class ToolUIErrorBoundary extends React.Component<
  ToolUIErrorBoundaryProps,
  ToolUIErrorBoundaryState
> {
  constructor(props: ToolUIErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): ToolUIErrorBoundaryState {
    return { hasError: true };
  }

  componentDidUpdate(prevProps: ToolUIErrorBoundaryProps) {
    if (prevProps.children !== this.props.children && this.state.hasError) {
      this.setState({ hasError: false });
    }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback;
    }
    return this.props.children;
  }
}
