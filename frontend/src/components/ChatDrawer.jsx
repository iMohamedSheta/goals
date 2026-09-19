import * as React from 'react';
import { Bot, Send, X, Square, Plus, Loader2, Search, ChevronUp, Check } from 'lucide-react';
import { cn } from '../lib/utils';
import {
  AIStatus, AIModels, AskAI, CancelAI, GetAIModel, SetAIModel,
} from '../../wailsjs/go/main/App';

/**
 * In-app AI chat side-drawer, powered by the local OpenCode CLI
 * (Go backend shells out to `opencode run --format json`).
 *
 * The assistant runs with the goals MCP server injected, so it can
 * read and manage tasks — after every reply we refresh the board
 * via onTasksChanged.
 */
export function ChatDrawer({ t, open, onClose, onTasksChanged }) {
  const [messages, setMessages] = React.useState([]); // {role:'user'|'ai', text}
  const [input, setInput] = React.useState('');
  const [sessionID, setSessionID] = React.useState('');
  const [model, setModel] = React.useState('');
  const [models, setModels] = React.useState([]);
  const [modelOpen, setModelOpen] = React.useState(false);
  const [modelQuery, setModelQuery] = React.useState('');
  const [status, setStatus] = React.useState(null);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState('');
  const bottomRef = React.useRef(null);
  const inputRef = React.useRef(null);
  const busyRef = React.useRef(false);
  const modelOpenRef = React.useRef(false);
  modelOpenRef.current = modelOpen;

  // load status + model once per open
  React.useEffect(() => {
    if (!open) return;
    let cancelled = false;
    (async () => {
      try {
        const [st, m, list] = await Promise.all([
          AIStatus().catch(() => null),
          GetAIModel().catch(() => ''),
          AIModels().catch(() => []),
        ]);
        if (cancelled) return;
        setStatus(st);
        setModel(m || '');
        setModels(Array.isArray(list) ? list : []);
      } catch {
        /* offline — banner shows on first send */
      }
      setTimeout(() => inputRef.current?.focus(), 60);
    })();
    return () => { cancelled = true; };
  }, [open ]);

  React.useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [messages, busy]);

  // Escape closes the model picker first, then the drawer (when idle)
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e) => {
      if (e.key !== 'Escape') return;
      if (modelOpenRef.current) {
        setModelOpen(false);
        setModelQuery('');
      } else if (!busyRef.current) {
        onClose?.();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;

  const ready = status?.available !== false;
  const shortModel = model ? model.split('/').pop() : t.aiAuto;
  const q = modelQuery.trim().toLowerCase();
  const filtered = (models || []).filter((m) => !q || m.toLowerCase().includes(q));
  const showAuto = !q || t.aiAuto.toLowerCase().includes(q) || 'auto'.includes(q);

  const send = async () => {
    const prompt = input.trim();
    if (!prompt || busyRef.current) return;
    setModelOpen(false);
    setError('');
    setInput('');
    setMessages((ms) => [...ms, { role: 'user', text: prompt }]);
    busyRef.current = true;
    setBusy(true);
    try {
      const res = await AskAI({ prompt, sessionID, model });
      if (res?.sessionID) setSessionID(res.sessionID);
      if (!model && res?.model) {
        setModel(res.model);
        try { await SetAIModel(res.model); } catch { /* noop */ }
      }
      setMessages((ms) => [...ms, { role: 'ai', text: res?.reply || '' }]);
      // the assistant may have changed tasks through MCP — refresh the board
      try { await onTasksChanged?.(); } catch { /* noop */ }
    } catch (e) {
      const msg = String(e?.message || e || '');
      if (/cancelled/i.test(msg)) {
        setMessages((ms) => [...ms, { role: 'ai', text: t.aiCancelled }]);
      } else {
        setError(msg);
      }
    } finally {
      busyRef.current = false;
      setBusy(false);
      setTimeout(() => inputRef.current?.focus(), 30);
    }
  };

  const stop = async () => {
    try { await CancelAI(); } catch { /* noop */ }
  };

  const newChat = () => {
    if (busyRef.current) return;
    setMessages([]);
    setSessionID('');
    setError('');
    setInput('');
    setTimeout(() => inputRef.current?.focus(), 30);
  };

  const pickModel = async (v) => {
    setModel(v);
    setModelOpen(false);
    setModelQuery('');
    try { await SetAIModel(v); } catch { /* local-only fallback */ }
    setTimeout(() => inputRef.current?.focus(), 30);
  };

  const renderModelName = (m) => {
    const i = m.indexOf('/');
    if (i < 0) return <span className="truncate">{m}</span>;
    return (
      <span className="min-w-0 flex-1 truncate">
        <span className="text-muted-foreground">{m.slice(0, i + 1)}</span>
        <span className="font-medium">{m.slice(i + 1)}</span>
      </span>
    );
  };

  return (
    <div className="fixed inset-0 z-50">
      <div className="absolute inset-0 bg-black/40 animate-fade-in" onClick={() => !busyRef.current && onClose?.()} />
      <aside className="absolute inset-y-0 end-0 flex w-full max-w-md flex-col border-s bg-background shadow-2xl animate-slide-in">
        {/* header */}
        <div className="flex shrink-0 items-center gap-2.5 border-b px-4 py-3">
          <span className="grid size-9 place-items-center rounded-xl bg-primary/15 text-primary">
            <Bot size={18} />
          </span>
          <div className="min-w-0 flex-1">
            <h3 className="truncate text-sm font-bold">{t.aiTitle}</h3>
            <p className="truncate text-[11px] text-muted-foreground">
              {status?.version ? `opencode v${status.version}` : t.aiPowered}
              {sessionID ? ' · ' + t.aiSession : ''}
            </p>
          </div>
          <button
            onClick={newChat}
            disabled={busy}
            title={t.aiNew}
            className="rounded-lg p-2 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:opacity-40"
          >
            <Plus size={16} />
          </button>
          <button
            onClick={() => !busyRef.current && onClose?.()}
            title={t.close}
            className="rounded-lg p-2 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <X size={16} />
          </button>
        </div>

        {/* setup warnings */}
        {status && !status.available && (
          <div className="mx-3 mt-3 shrink-0 rounded-xl border border-red-500/30 bg-red-500/10 px-3.5 py-2.5 text-xs leading-relaxed text-red-300" dir="ltr">
            {status.hint || t.aiNeedSetup}
          </div>
        )}
        {status?.available && !status.authenticated && (
          <div className="mx-3 mt-3 shrink-0 rounded-xl border border-amber-500/30 bg-amber-500/10 px-3.5 py-2.5 text-xs leading-relaxed" dir="ltr">
            {status.hint || t.aiAuthNeeded}
          </div>
        )}

        {/* messages */}
        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto px-4 py-4">
          {messages.length === 0 && (
            <div className="flex h-full flex-col items-center justify-center gap-2.5 text-center">
              <span className="grid size-12 place-items-center rounded-2xl bg-primary/10 text-primary">
                <Bot size={24} />
              </span>
              <p className="max-w-[30ch] text-[13px] leading-relaxed text-muted-foreground">{t.aiEmpty}</p>
            </div>
          )}
          {messages.map((m, i) => (
            <div key={i} className={cn('flex', m.role === 'user' ? 'justify-end' : 'justify-start')}>
              <div
                className={cn(
                  'max-w-[85%] whitespace-pre-wrap break-words rounded-2xl px-3.5 py-2.5 text-[13px] leading-relaxed shadow-sm',
                  m.role === 'user'
                    ? 'rounded-br-md bg-primary text-primary-foreground'
                    : 'rounded-bl-md border bg-card'
                )}
              >
                {m.text}
              </div>
            </div>
          ))}
          {busy && (
            <div className="flex justify-start">
              <div className="flex items-center gap-2 rounded-2xl rounded-bl-md border bg-card px-3.5 py-2.5 text-[13px] text-muted-foreground shadow-sm">
                <Loader2 size={14} className="animate-spin" />
                {t.aiThinking}
              </div>
            </div>
          )}
          {error && (
            <div className="rounded-xl border border-red-500/30 bg-red-500/10 px-3.5 py-2.5 text-xs leading-relaxed text-red-300" dir="ltr">
              {error}
            </div>
          )}
          <div ref={bottomRef} />
        </div>

        {/* composer */}
        <div className="relative shrink-0 p-3 pt-1">
          {modelOpen && (
            <>
              <div className="fixed inset-0 z-10" onClick={() => { setModelOpen(false); setModelQuery(''); }} />
              <div className="absolute inset-x-3 bottom-full z-20 mb-2 overflow-hidden rounded-2xl border bg-popover shadow-2xl animate-slide-in">
                <div className="relative border-b border-border/60">
                  <Search size={14} className="pointer-events-none absolute start-3.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
                  <input
                    autoFocus
                    value={modelQuery}
                    onChange={(e) => setModelQuery(e.target.value)}
                    placeholder={t.aiSearchModels}
                    dir="ltr"
                    className="w-full bg-transparent py-3 pe-3.5 ps-10 text-left text-[13px] outline-none placeholder:text-muted-foreground"
                  />
                </div>
                <div className="max-h-56 overflow-y-auto p-1.5" dir="ltr">
                  {showAuto && (
                    <button
                      onClick={() => pickModel('')}
                      className={cn(
                        'flex w-full items-center gap-2.5 rounded-xl px-3 py-2.5 text-left text-[13px] transition-colors hover:bg-accent',
                        model === '' && 'bg-accent/60'
                      )}
                    >
                      <span className="grid size-6 shrink-0 place-items-center rounded-full bg-primary/15 text-primary">
                        <Bot size={13} />
                      </span>
                      <span className="flex-1 truncate font-semibold">{t.aiAuto}</span>
                      {model === '' && <Check size={15} className="shrink-0 text-primary" />}
                    </button>
                  )}
                  {filtered.map((m) => (
                    <button
                      key={m}
                      onClick={() => pickModel(m)}
                      className={cn(
                        'flex w-full items-center gap-2.5 rounded-xl px-3 py-2.5 text-left text-[13px] tabular transition-colors hover:bg-accent',
                        model === m && 'bg-accent/60'
                      )}
                    >
                      {renderModelName(m)}
                      {model === m && <Check size={15} className="shrink-0 text-primary" />}
                    </button>
                  ))}
                  {!showAuto && filtered.length === 0 && (
                    <p className="px-3 py-4 text-center text-xs text-muted-foreground">{t.aiNoModels}</p>
                  )}
                </div>
              </div>
            </>
          )}

          <div className="rounded-2xl border border-input bg-card shadow-sm transition-colors focus-within:border-primary/60 focus-within:ring-2 focus-within:ring-primary/15">
            <textarea
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); }
              }}
              placeholder={t.aiPh}
              rows={2}
              disabled={!ready}
              className="max-h-32 min-h-[52px] w-full resize-none bg-transparent px-4 pt-3 text-[13px] leading-relaxed outline-none placeholder:text-muted-foreground disabled:opacity-50"
            />
            <div className="flex items-center gap-2 px-2.5 pb-2.5">
              <button
                onClick={() => { setModelOpen((v) => !v); setModelQuery(''); }}
                title={t.aiModel}
                dir="ltr"
                className={cn(
                  'flex h-8 max-w-[160px] items-center gap-1.5 rounded-full px-3 text-[11px] font-semibold transition-colors',
                  modelOpen ? 'bg-primary/15 text-primary' : 'bg-muted text-muted-foreground hover:bg-accent hover:text-foreground'
                )}
              >
                <ChevronUp size={13} className={cn('shrink-0 transition-transform', modelOpen && 'rotate-180')} />
                <span className="truncate tabular">{shortModel}</span>
              </button>
              <span className="flex-1" />
              {busy ? (
                <button
                  onClick={stop}
                  title={t.aiStop}
                  className="grid size-9 shrink-0 place-items-center rounded-full bg-secondary text-secondary-foreground transition-colors hover:bg-secondary/80"
                >
                  <Square size={14} />
                </button>
              ) : (
                <button
                  onClick={send}
                  disabled={!input.trim() || !ready}
                  title={t.aiSend}
                  className="grid size-9 shrink-0 place-items-center rounded-full bg-primary text-primary-foreground shadow transition-all hover:brightness-110 active:scale-95 disabled:pointer-events-none disabled:opacity-40"
                >
                  <Send size={15} />
                </button>
              )}
            </div>
          </div>
          <p className="mt-1.5 text-center text-[10px] text-muted-foreground">{t.aiHint}</p>
        </div>
      </aside>
    </div>
  );
}
