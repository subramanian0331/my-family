import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("App error:", error, info);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex min-h-screen items-center justify-center bg-slate-50 px-4">
          <div className="max-w-md rounded-2xl border border-red-200 bg-white p-8 text-center shadow-sm">
            <h1 className="text-lg font-semibold text-slate-900">Something went wrong</h1>
            <p className="mt-2 text-sm text-red-600">{this.state.error.message}</p>
            <button
              onClick={() => window.location.assign("/login")}
              className="mt-6 rounded-lg bg-accent px-4 py-2 text-white hover:bg-accent-hover"
            >
              Go to login
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}