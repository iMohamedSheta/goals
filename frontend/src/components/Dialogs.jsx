import * as React from 'react';
import { Star, History } from 'lucide-react';
import { cn } from '../lib/utils';
import { Button } from './ui/button';
import { Sheet, SheetHeader, SheetBody, SheetFooter } from './ui/sheet';
import { Input, Textarea, Select, Label } from './ui/form';
import { STATUSES } from './Board';
import { formatHMS } from '../lib/i18n';

function todayISO() {
  return new Date().toISOString().slice(0, 10);
}
function plusDaysISO(days) {
  const d = new Date();
  d.setDate(d.getDate() + (days || 7));
  return d.toISOString().slice(0, 10);
}

function Field({ label, children }) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      {children}
    </div>
  );
}

export function TaskSheet({ t, open, onClose, task, horizons, contexts, activeHorizon, presetStatus, presetParentId, tasks: allTasks, onSave, entries }) {
  const hz = horizons.find((h) => h.key === (task?.horizon || activeHorizon));
  const [form, setForm] = React.useState(null);

  // Possible parents: same-horizon tasks excluding self + descendants (cycle guard).
  const parentOptions = React.useMemo(() => {
    const list = allTasks || [];
    if (!open) return [];
    const horizonKey = form?.horizon || task?.horizon || activeHorizon || 'short';
    const banned = new Set(task ? [task.id] : []);
    if (task) {
      const stack = (list.filter((x) => x.parentId === task.id) || []).map((x) => x.id);
      while (stack.length) {
        const cur = stack.pop();
        if (banned.has(cur)) continue;
        banned.add(cur);
        for (const kid of list.filter((x) => x.parentId === cur)) stack.push(kid.id);
      }
    }
    return list
      .filter((x) => (!task || x.id !== task.id) && !banned.has(x.id))
      .filter((x) => (x.horizon || activeHorizon) === horizonKey || (!task && x.horizon === (form?.horizon || activeHorizon)))
      .sort((a, b) => String(a.title).localeCompare(String(b.title)));
  }, [open, allTasks, task, form?.horizon, activeHorizon]); // eslint-disable-line

  React.useEffect(() => {
    if (!open) return;
    setForm({
      title: task?.title || '',
      description: task?.description || '',
      horizon: task?.horizon || activeHorizon || 'short',
      status: task?.status || presetStatus || 'todo',
      contextId: task?.contextId || '',
      parentId: task?.parentId || presetParentId || '',
      priority: task?.priority || 'medium',
      startDate: task?.startDate || todayISO(),
      dueDate: task?.dueDate || plusDaysISO(hz?.defaultDays || 7),
      focus: task?.focus || false,
      maxH: Math.floor((task?.maxSeconds || 0) / 3600),
      maxM: Math.floor(((task?.maxSeconds || 0) % 3600) / 60),
    });
  }, [open]); // eslint-disable-line

  if (!open || !form) return null;
  const set = (k, v) => setForm((f) => ({ ...f, [k]: v }));
  const valid = form.title.trim().length > 0;

  return (
    <Sheet open={open} onClose={onClose}>
      <SheetHeader
        title={task ? t.editTaskTitle : t.newTaskTitle}
        description={task ? t.editTaskDesc : t.newTaskDesc}
        onClose={onClose}
      />
      <SheetBody>
        <Field label={t.titleLabel}>
          <Input autoFocus placeholder={t.titlePh} value={form.title} onChange={(e) => set('title', e.target.value)} />
        </Field>
        <Field label={t.descLabel}>
          <Textarea placeholder={t.descPh} value={form.description} onChange={(e) => set('description', e.target.value)} />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label={t.horizon}>
            <Select value={form.horizon} onChange={(e) => {
              const h = horizons.find((x) => x.key === e.target.value);
              setForm((f) => ({ ...f, horizon: e.target.value, dueDate: task?.dueDate || plusDaysISO(h?.defaultDays || 7) }));
            }}>
              {horizons.map((h) => <option key={h.key} value={h.key}>{h.label} (~{h.defaultDays})</option>)}
            </Select>
          </Field>
          <Field label={t.status}>
            <Select value={form.status} onChange={(e) => set('status', e.target.value)}>
              {STATUSES.map((s) => <option key={s.key} value={s.key}>{t[s.key]}</option>)}
            </Select>
          </Field>
          <Field label={t.context}>
            <Select value={form.contextId} onChange={(e) => set('contextId', e.target.value)}>
              <option value="">{t.none}</option>
              {contexts.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          </Field>
          <Field label={t.parentTask || 'Parent task'}>
            <Select
              value={form.parentId}
              onChange={(e) => {
                const pid = e.target.value;
                if (pid) {
                  const p = (allTasks || []).find((x) => x.id === pid);
                  if (p?.horizon) {
                    setForm((f) => ({ ...f, parentId: pid, horizon: p.horizon }));
                    return;
                  }
                }
                set('parentId', pid);
              }}
            >
              <option value="">{t.topLevel || '— Top level —'}</option>
              {parentOptions.map((p) => <option key={p.id} value={p.id}>{p.title}</option>)}
            </Select>
          </Field>
          <Field label={t.priority}>
            <Select value={form.priority} onChange={(e) => set('priority', e.target.value)}>
              {['low', 'medium', 'high', 'urgent'].map((p) => <option key={p} value={p}>{t[p]}</option>)}
            </Select>
          </Field>
          <Field label={t.start}>
            <Input type="date" value={form.startDate} onChange={(e) => set('startDate', e.target.value)} />
          </Field>
          <Field label={t.due}>
            <Input type="date" value={form.dueDate} onChange={(e) => set('dueDate', e.target.value)} />
          </Field>
        </div>
        <div>
          <Label>{t.maxTime} <span className="font-normal text-muted-foreground">· {t.noMax} = 0</span></Label>
          <div className="mt-2 flex items-center gap-2">
            <Input
              type="number" min={0} max={999} value={form.maxH}
              onChange={(e) => set('maxH', Math.max(0, parseInt(e.target.value, 10) || 0))}
              className="w-20 text-center tabular"
            />
            <span className="text-xs text-muted-foreground">{t.hoursShort}</span>
            <Input
              type="number" min={0} max={59} value={form.maxM}
              onChange={(e) => set('maxM', Math.min(59, Math.max(0, parseInt(e.target.value, 10) || 0)))}
              className="w-20 text-center tabular"
            />
            <span className="text-xs text-muted-foreground">{t.minutesShort}</span>
          </div>
        </div>
        <button
          onClick={() => set('focus', !form.focus)}
          className={cn(
            'flex w-full items-center gap-2.5 rounded-lg border px-3.5 py-2.5 text-sm font-medium transition-colors',
            form.focus ? 'border-amber-500/40 bg-amber-500/10 text-amber-300' : 'border-input text-muted-foreground hover:bg-accent hover:text-foreground'
          )}
        >
          <Star size={16} fill={form.focus ? 'currentColor' : 'none'} />
          {t.pinFocus}
          <span className="ms-auto text-xs opacity-70">{form.focus ? t.pinned : t.off}</span>
        </button>

        {task && (
          <div className="rounded-lg border p-3.5">
            <div className="mb-2 flex items-center gap-2 text-sm font-semibold">
              <History size={15} className="text-muted-foreground" />
              {t.timeLog}
              <span className="tabular ms-auto text-xs font-bold text-emerald-300">{formatHMS(task.elapsedSeconds || 0)}</span>
            </div>
            {(entries || []).length === 0 ? (
              <p className="text-xs leading-relaxed text-muted-foreground">{t.noSessions}</p>
            ) : (
              <div className="max-h-36 space-y-1 overflow-y-auto">
                {(entries || []).map((e) => (
                  <div key={e.id} className="tabular flex items-center justify-between text-xs text-muted-foreground">
                    <span>{(e.startedAt || '').slice(0, 16).replace('T', ' ')}</span>
                    <span className="font-semibold text-foreground">{formatHMS(e.seconds)}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </SheetBody>
      <SheetFooter>
        <Button variant="ghost" onClick={onClose}>{t.cancel}</Button>
        <Button
          disabled={!valid}
          onClick={() => onSave({
            title: form.title.trim(),
            description: form.description,
            horizon: form.horizon,
            status: form.status,
            contextId: form.contextId || null,
            parentId: form.parentId || null,
            priority: form.priority,
            startDate: form.startDate || null,
            dueDate: form.dueDate || null,
            focus: form.focus,
            maxSeconds: (form.maxH || 0) * 3600 + (form.maxM || 0) * 60,
          })}
        >
          {task ? t.save : t.create}
        </Button>
      </SheetFooter>
    </Sheet>
  );
}

export function ConfirmModal({ t, open, onClose, title, message, confirmLabel, danger, onConfirm }) {
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e) => e.key === 'Escape' && onClose?.();
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div
        className="absolute inset-0 bg-black/70 backdrop-blur-[2px] animate-fade-in"
        onMouseDown={(e) => e.target === e.currentTarget && onClose?.()}
      />
      <div className="animate-modal-pop relative w-full max-w-sm rounded-xl border bg-popover p-6 text-popover-foreground shadow-2xl">
        <h2 className="text-base font-bold tracking-tight">{title}</h2>
        <p className="mt-1.5 text-[13px] leading-relaxed text-muted-foreground">{message}</p>
        <div className="mt-5 flex items-center justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>{t.cancel}</Button>
          <Button variant={danger === false ? 'default' : 'destructive'} onClick={onConfirm}>{confirmLabel}</Button>
        </div>
      </div>
    </div>
  );
}
