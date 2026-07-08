"use client";

import React, { Component, ErrorInfo, ReactNode } from "react";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("ErrorBoundary caught an error:", error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="rounded-lg border border-red-800 bg-red-900/50 p-6">
          <h2 className="text-lg font-semibold text-red-200">
            Something went wrong
          </h2>
          <p className="mt-2 text-sm text-red-300">
            {this.state.error?.message || "An unexpected error occurred"}
          </p>
          <button
            onClick={() => this.setState({ hasError: false, error: null })}
            className="mt-4 rounded-md bg-red-800 px-4 py-2 text-sm text-white hover:bg-red-700"
          >
            Try again
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}
