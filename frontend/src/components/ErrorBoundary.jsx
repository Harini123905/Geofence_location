import { Component } from "react";

export default class ErrorBoundary extends Component {
  state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch(error) {
    console.error("Page crashed:", error);
  }

  render() {
    if (this.state.failed) {
      return (
        <section className="card">
          <h2>Something went wrong</h2>
          <p className="muted">This page could not be displayed. Please reload the page.</p>
          <button onClick={() => window.location.reload()}>Reload</button>
        </section>
      );
    }
    return this.props.children;
  }
}