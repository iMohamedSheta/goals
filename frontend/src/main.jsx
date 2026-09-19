import React from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import './index.css';
import { loadLocalAppearance, applyAppearance } from './lib/appearance';

// apply saved visuals before first paint (no flicker)
try {
  applyAppearance(loadLocalAppearance());
} catch { /* noop */ }

// forward boot/render errors to the backend log (debuggable via %TEMP%/goals-error.log)
function reportFrontendError(where, err) {
  try {
    const msg = String((err && (err.stack || err.message)) || err);
    window?.go?.main?.App?.ReportError?.(`${where}: ${msg.slice(0, 2000)}`).catch(() => {});
  } catch { /* bindings not ready yet */ }
}
window.addEventListener('error', (e) => reportFrontendError('window.onerror', e?.error || e?.message));
window.addEventListener('unhandledrejection', (e) => reportFrontendError('unhandledrejection', e?.reason));

// Never show a blank page: render the crash visibly instead.
class RootError extends React.Component {
  constructor(props) {
    super(props);
    this.state = { error: null };
  }
  static getDerivedStateFromError(e) {
    return { error: String(e?.message || e) };
  }
  componentDidCatch(e) {
    reportFrontendError('react-crash', e);
  }
  render() {
    if (this.state.error) {
      return (
        <div dir="rtl" style={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#09090f', color: '#e2e8f0', fontFamily: 'sans-serif' }}>
          <div style={{ maxWidth: 520, border: '1px solid #7f1d1d', background: '#450a0a33', borderRadius: 12, padding: 24, textAlign: 'center' }}>
            <h2 style={{ color: '#f87171', margin: '0 0 8px' }}>حدث خطأ — Something broke</h2>
            <p dir="ltr" style={{ fontSize: 12, color: '#94a3b8', wordBreak: 'break-word' }}>{this.state.error}</p>
            <p style={{ fontSize: 12, color: '#94a3b8' }}>التفاصيل في %TEMP%\goals-error.log</p>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}

createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <RootError>
      <App />
    </RootError>
  </React.StrictMode>
);
