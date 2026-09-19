import * as React from 'react';
import {
  Circle, Timer as TimerIcon, AlertCircle, CheckCircle2, Star, Plus, Pencil, Trash2,
  CalendarClock, LayoutGrid, List as ListIcon, SearchX, Crosshair, Minus, Square, X,
  Play, Pause, Check, PictureInPicture2, Maximize2, Minimize2, ChevronsUp,
} from 'lucide-react';
import { cn } from '../lib/utils';
import { Badge } from './ui/badge';
import { Button } from './ui/button';
import { formatHMS, liveElapsed } from '../lib/i18n';
import { WindowHide, WindowMinimise } from '../../wailsjs/runtime/runtime';

/** Compact normal-style caption buttons for the mini window. */
function MiniCap({ t, onExpand }) {
  const cls = 'flex h-8 w-10 items-center justify-center text-muted-foreground transition-colors hover:bg-accent hover:text-foreground';
  return (
    <>
      <button title={t.winMin} onClick={() => WindowMinimise()} className={cls}><Minus size={13} /></button>
      <button title={t.expand} onClick={onExpand} className={cls}><Maximize2 size={12} /></button>
      <button title={t.winHide} onClick={() => WindowHide()} className={cn(cls, 'hover:bg-[#E81123] hover:text-white')}><X size={14} /></button>
    </>
  );
}

export const STATUSES = [
  { key: 'todo', icon: Circle },
  { key: 'in_progress', icon: TimerIcon },
  { key: 'blocked', icon: AlertCircle },
  { key: 'done', icon: CheckCircle2 },
];

const PRIORITY_LABEL = 'border-border bg-secondary text-muted-foreground';

/** Self-ticking time label — re-renders only itself, safe during drag & drop. */
export function LiveTime({ task, elapsed: elapsedProp, max, className, overClassName, showBadge, badgeLabel }) {
  const [, setNow] = React.useState(Date.now());
  React.useEffect(() => {
    if (!task?.timerStartedAt) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [task?.timerStartedAt]);
  const live = task?.timerStartedAt
    ? (task.elapsedSeconds || 0) + Math.max(0, Math.floor((Date.now() - Date.parse(task.timerStartedAt)) / 1000))
    : task?.elapsedSeconds ?? task?.elapsed ?? 0;
  const secs = elapsedProp ?? live;
  const over = (max || task?.maxSeconds || 0) > 0 && secs > (max || task?.maxSeconds || 0);
  return (
    <>
      <span className={cn('tabular', className, over && (overClassName || 'text-red-400'))}>{formatHMS(secs)}</span>
      {showBadge && over && (
        <span className="inline-flex shrink-0 items-center rounded-md border border-red-500/40 bg-red-500/10 px-1.5 py-0.5 text-[10px] font-bold text-red-400">
          {badgeLabel}
        </span>
      )}
    </>
  );
}

export function dueInfo(task, t) {
  if (!task.dueDate) return null;
  const neutral = 'border-border bg-secondary text-muted-foreground';
  if (task.status === 'done') return { label: task.dueDate, cls: neutral };
  const today = new Date().toISOString().slice(0, 10);
  if (task.dueDate < today) return { label: `${task.dueDate} · ${t.overdue}`, cls: 'border-red-500/30 bg-red-500/10 text-red-400' };
  if (task.dueDate === today) return { label: `${task.dueDate} · ${t.today}`, cls: neutral };
  return { label: task.dueDate, cls: neutral };
}

export function ContextBadge({ name, color }) {
  if (!name) return <span className="text-xs text-muted-foreground">—</span>;
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-secondary px-2 py-0.5 text-[11px] font-medium text-secondary-foreground">
      <span className="size-1.5 rounded-full" style={{ background: color }} />
      <span className="max-w-[110px] truncate">{name}</span>
    </span>
  );
}

function IconBtn({ title, onClick, className, children }) {
  return (
    <button
      title={title}
      onClick={(e) => { e.stopPropagation(); onClick?.(); }}
      className={cn('rounded-md p-1.5 transition-colors', className)}
    >
      {children}
    </button>
  );
}

/** Timer row rendered under every card + table row actions. */
export function TimerControls({ t, task, onStart, onPause, onFinish }) {
  const running = !!task.timerStartedAt;
  const hasTime = running || (task.elapsedSeconds || 0) > 0;
  return (
    <div
      className={cn(
        'flex items-center gap-1.5 rounded-lg border px-2 py-1.5',
        running ? 'border-emerald-500/40 bg-emerald-500/10' : 'border-border/70 bg-muted/30'
      )}
      onClick={(e) => e.stopPropagation()}
    >
      {running ? (
        <span className="pulse-dot ms-1 size-2 shrink-0 rounded-full bg-emerald-400 text-emerald-400" />
      ) : (
        <TimerIcon size={13} className="ms-1 shrink-0 text-muted-foreground" />
      )}
      <LiveTime task={task} className={cn('text-xs font-bold', running ? 'text-emerald-300' : 'text-muted-foreground')} showBadge badgeLabel={t.overtime} />
      <span className="flex-1" />
      {!running ? (
        <IconBtn title={hasTime ? t.resumeTimer : t.startTimer} onClick={() => onStart(task)} className="text-muted-foreground hover:bg-emerald-500/15 hover:text-emerald-300">
          <Play size={14} />
        </IconBtn>
      ) : (
        <IconBtn title={t.pauseTimer} onClick={() => onPause(task)} className="text-emerald-300 hover:bg-emerald-500/20">
          <Pause size={14} />
        </IconBtn>
      )}
      <IconBtn title={t.finishTask} onClick={() => onFinish(task)} className="text-muted-foreground hover:bg-primary/15 hover:text-primary">
        <Check size={14} />
      </IconBtn>
    </div>
  );
}

const TaskCard = React.memo(function TaskCard({ t, task, context, onEdit, onDelete, onToggleFocus, onStart, onPause, onFinish }) {
  const [dragging, setDragging] = React.useState(false);
  const due = dueInfo(task, t);
  return (
    <div
      draggable
      onDragStart={(e) => {
        e.dataTransfer.setData('text/task-id', task.id);
        e.dataTransfer.effectAllowed = 'move';
        setDragging(true);
      }}
      onDragEnd={() => setDragging(false)}
      onClick={() => onEdit(task)}
      className={cn(
        'tcard surf group relative h-auto min-h-[190px] w-full cursor-pointer rounded-xl border bg-card p-4 shadow-sm transition-all hover:-translate-y-px hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5',
        'shrink-0 grow-0 basis-auto', // natural height — never squish, column scrolls instead
        task.focus && 'border-amber-500/40',
        task.timerStartedAt && 'border-emerald-500/40',
        dragging && 'dragging-card'
      )}
    >
      <div className="flex min-w-0 flex-col">
        <div className="flex items-center gap-1.5">
          <div className="flex min-w-0 flex-1 items-center gap-1.5">
            <ContextBadge name={context?.name || task.contextName} color={context?.color || task.contextColor || '#6366f1'} />
            {task.focus && (
              <span className="inline-flex shrink-0 items-center gap-1 rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[11px] font-semibold text-amber-400">
                <Star size={10} fill="currentColor" /> {t.focusBadge}
              </span>
            )}
          </div>
          <button
            onClick={(e) => { e.stopPropagation(); onToggleFocus(task); }}
            className={cn('shrink-0 rounded p-1 transition-colors', task.focus ? 'text-amber-400' : 'text-muted-foreground opacity-0 hover:text-amber-400 group-hover:opacity-100')}
            title="Focus"
          >
            <Star size={15} fill={task.focus ? 'currentColor' : 'none'} />
          </button>
        </div>

        <h4 className="mt-2.5 break-words text-sm font-semibold leading-relaxed">{task.title}</h4>
        {task.description && <p className="cdesc mt-1.5 whitespace-pre-wrap break-words text-xs leading-relaxed text-muted-foreground">{task.description}</p>}

        <div className="mt-3 flex flex-wrap items-center gap-1.5">
          <span className={cn('inline-flex items-center rounded-md border px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide', PRIORITY_LABEL)}>
            {t[task.priority] || task.priority}
          </span>
          {due && (
            <span className={cn('inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[11px] font-medium tabular', due.cls)}>
              <CalendarClock size={11} /> {due.label}
            </span>
          )}
        </div>

        <div className="mt-2.5">
          <TimerControls t={t} task={task} onStart={onStart} onPause={onPause} onFinish={onFinish} />
        </div>

        <div className="mt-1 flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
          <button
            onClick={(e) => { e.stopPropagation(); onEdit(task); }}
            className="flex items-center gap-1 rounded-md px-1.5 py-1 text-[11px] font-medium text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <Pencil size={12} /> {t.edit}
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); onDelete(task); }}
            className="flex items-center gap-1 rounded-md px-1.5 py-1 text-[11px] font-medium text-muted-foreground hover:bg-destructive/10 hover:text-red-400"
          >
            <Trash2 size={12} /> {t.delete}
          </button>
        </div>
      </div>
    </div>
  );
});

const Column = React.memo(function Column({ t, status, tasks, contextOf, onEdit, onDelete, onToggleFocus, onStart, onPause, onFinish, onDrop }) {
  const [over, setOver] = React.useState(false);
  const meta = STATUSES.find((s) => s.key === status) || STATUSES[0];
  const Icon = meta.icon;
  return (
    <div className="surf flex h-full max-h-full min-h-0 w-72 shrink-0 grow-0 basis-72 flex-col rounded-xl border bg-card/40">
      <div className="flex shrink-0 items-center gap-2 px-3.5 pb-2.5 pt-3.5">
        <Icon size={15} className="text-muted-foreground" />
        <span className="text-[13px] font-semibold">{t[status]}</span>
        <span className="tabular ms-auto rounded-full bg-secondary px-2 py-0.5 text-[11px] font-semibold text-muted-foreground">
          {tasks.length}
        </span>
      </div>
      <div
        onDragOver={(e) => { e.preventDefault(); setOver(true); }}
        onDragLeave={() => setOver(false)}
        onDrop={(e) => {
          e.preventDefault();
          setOver(false);
          const id = e.dataTransfer.getData('text/task-id');
          if (id) onDrop(id, status);
        }}
        className={cn('colbody thin-scroll mx-2 mb-2 flex min-h-0 flex-1 flex-col gap-2.5 overflow-y-auto overflow-x-hidden rounded-lg p-2 transition-colors', over && 'drop-target')}
      >
        {tasks.map((task) => (
          <TaskCard key={task.id} t={t} task={task} context={contextOf(task.contextId)} onEdit={onEdit} onDelete={onDelete} onToggleFocus={onToggleFocus} onStart={onStart} onPause={onPause} onFinish={onFinish} />
        ))}
        {tasks.length === 0 && (
          <div className="shrink-0 rounded-lg border border-dashed border-border px-3 py-6 text-center text-xs text-muted-foreground">
            {t.dropHere}
          </div>
        )}
      </div>
    </div>
  );
});

export function KanbanBoard(props) {
  const { t, tasks, onCreateFirst } = props;
  if (tasks.length === 0) {
    return (
      <div className="flex flex-1 flex-col bg-white! items-center justify-center gap-3 py-20 text-center animate-slide-in">
        <div className="flex size-14 items-center justify-center rounded-2xl border bg-card shadow-sm">
          <Crosshair size={24} className="text-primary" />
        </div>
        <div>
          <h3 className="text-base font-semibold">{t.noMatch}</h3>
          <p className="mt-1 text-sm text-muted-foreground">{t.noMatchSub}</p>
        </div>
        <Button onClick={onCreateFirst}><Plus /> {t.newTask}</Button>
      </div>
    );
  }
  return (
    <div className="flex h-full min-h-0 flex-1 items-stretch gap-3 overflow-x-auto overflow-y-hidden pb-2">
      {STATUSES.map((s) => (
        <Column key={s.key} {...props} status={s.key} tasks={tasks.filter((x) => x.status === s.key)} />
      ))}
    </div>
  );
}

export function TaskTable({ t, tasks, contextOf, onEdit, onDelete, onToggleFocus, onStart, onPause, onFinish }) {
  if (tasks.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 py-20 text-center">
        <div className="flex size-14 items-center justify-center rounded-2xl border bg-card shadow-sm">
          <SearchX size={24} className="text-muted-foreground" />
        </div>
        <h3 className="text-base font-semibold">{t.nothingHere}</h3>
        <p className="text-sm text-muted-foreground">{t.nothingSub}</p>
      </div>
    );
  }
  return (
    <div className="surf overflow-hidden rounded-xl border bg-card shadow-sm">
      <div className="grid grid-cols-[28px_minmax(0,1fr)_110px_80px_120px_110px_90px_60px] items-center gap-2 border-b bg-muted/40 px-4 py-2.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
        <span />
        <span>{t.task}</span><span>{t.context}</span><span>{t.priority}</span><span>{t.totalTime}</span><span>{t.due}</span><span>{t.status}</span><span />
      </div>
      {tasks.map((task) => {
        const ctx = contextOf(task.contextId);
        const due = dueInfo(task, t);
        const sm = STATUSES.find((s) => s.key === task.status) || STATUSES[0];
        const SIcon = sm.icon;
        const running = !!task.timerStartedAt;
        return (
          <div
            key={task.id}
            onClick={() => onEdit(task)}
            className="grid cursor-pointer grid-cols-[28px_minmax(0,1fr)_110px_80px_120px_110px_90px_60px] items-center gap-2 border-b px-4 py-2.5 transition-colors last:border-0 hover:bg-accent/50"
          >
            <button
              onClick={(e) => { e.stopPropagation(); onToggleFocus(task); }}
              className={task.focus ? 'text-amber-400' : 'text-muted-foreground hover:text-amber-400'}
            >
              <Star size={15} fill={task.focus ? 'currentColor' : 'none'} />
            </button>
            <span className="min-w-0">
              <span className={cn('block truncate text-[13.5px] font-semibold', task.status === 'done' && 'text-muted-foreground line-through')}>
                {task.title}
              </span>
              {task.description && <span className="block truncate text-xs text-muted-foreground">{task.description}</span>}
            </span>
            <span className="min-w-0"><ContextBadge name={ctx?.name || task.contextName} color={ctx?.color || task.contextColor || '#6366f1'} /></span>
            <span>
              <span className={cn('inline-flex rounded-md border px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide', PRIORITY_LABEL)}>
                {t[task.priority] || task.priority}
              </span>
            </span>
            <span className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
              {running && <span className="pulse-dot size-1.5 shrink-0 rounded-full bg-emerald-400 text-emerald-400" />}
              <LiveTime task={task} className={cn('text-xs font-bold', running ? 'text-emerald-300' : 'text-muted-foreground')} showBadge badgeLabel={t.overtime} />
              {!running ? (
                <button title={t.startTimer} onClick={() => onStart(task)} className="rounded p-1 text-muted-foreground hover:bg-emerald-500/15 hover:text-emerald-300"><Play size={13} /></button>
              ) : (
                <button title={t.pauseTimer} onClick={() => onPause(task)} className="rounded p-1 text-emerald-300 hover:bg-emerald-500/20"><Pause size={13} /></button>
              )}
              <button title={t.finishTask} onClick={() => onFinish(task)} className="rounded p-1 text-muted-foreground hover:bg-primary/15 hover:text-primary"><Check size={13} /></button>
            </span>
            <span className="text-xs tabular">
              {due ? <span className={cn('inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5', due.cls)}><CalendarClock size={11} />{due.label}</span> : <span className="text-muted-foreground">—</span>}
            </span>
            <span>
              <Badge variant="secondary">
                <SIcon size={12} /> {t[task.status]}
              </Badge>
            </span>
            <span className="flex justify-end gap-0.5">
              <button onClick={(e) => { e.stopPropagation(); onEdit(task); }} className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"><Pencil size={14} /></button>
              <button onClick={(e) => { e.stopPropagation(); onDelete(task); }} className="rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-red-400"><Trash2 size={14} /></button>
            </span>
          </div>
        );
      })}
    </div>
  );
}

export function ViewToggle({ t, view, onChange }) {
  return (
    <div className="flex rounded-lg border bg-secondary/60 p-0.5">
      {[
        { key: 'kanban', label: t.board, icon: LayoutGrid },
        { key: 'list', label: t.list, icon: ListIcon },
      ].map((v) => (
        <button
          key={v.key}
          onClick={() => onChange(v.key)}
          className={cn(
            'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-[13px] font-medium transition-all',
            view === v.key ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
          )}
        >
          <v.icon size={14} /> {v.label}
        </button>
      ))}
    </div>
  );
}

/** Header pill shown while a timer runs: live time + pause + mini + finish. */
export function ActiveTimerPill({ t, active, onPause, onResume, onMini, onFinish }) {
  if (!active) return null;
  const running = !!active.timerStartedAt;
  return (
    <div className="flex items-center gap-2 rounded-lg border border-emerald-500/40 bg-emerald-500/10 py-1 pe-1 ps-3 animate-slide-in">
      <span className={running ? 'pulse-dot size-2 rounded-full bg-emerald-400 text-emerald-400' : 'size-2 rounded-full bg-amber-400'} />
      <span className="max-w-[180px] truncate text-[13px] font-semibold text-emerald-200">{active.title}</span>
      <LiveTime task={active} elapsed={active.elapsed} max={active.maxSeconds} className="text-[13px] font-bold tabular text-emerald-300" showBadge badgeLabel={t.overtime} />
      {running ? (
        <button title={t.pauseTimer} onClick={onPause} className="rounded-md p-1.5 text-emerald-300 hover:bg-emerald-500/20"><Pause size={14} /></button>
      ) : (
        <button title={t.resumeTimer} onClick={onResume} className="rounded-md p-1.5 text-emerald-300 hover:bg-emerald-500/20"><Play size={14} /></button>
      )}
      <button title={t.miniMode} onClick={onMini} className="rounded-md p-1.5 text-emerald-300/80 hover:bg-emerald-500/20 hover:text-emerald-200"><PictureInPicture2 size={14} /></button>
      <button title={t.finishTask} onClick={onFinish} className="rounded-md bg-emerald-500/20 p-1.5 text-emerald-200 hover:bg-emerald-500/35"><Check size={14} /></button>
    </div>
  );
}

/** Full-window mini mode: only the timer. Window is small + always on top. */
export function MiniTimer({ t, active, onPause, onResume, onFinish, onExpand, onCollapse }) {
  return (
    <div className="flex h-full flex-col bg-background">
      <div className="flex h-8 shrink-0 select-none items-center border-b px-1" style={{ ['--wails-draggable']: 'drag' }}>
        <span className="pointer-events-none px-2 text-[11px] font-bold text-muted-foreground">Goals · {t.activeTimer}</span>
        <span className="ms-auto flex items-stretch" style={{ ['--wails-draggable']: 'no-drag' }}>
          <MiniCap t={t} onExpand={onExpand} />
        </span>
      </div>
      <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
        {!active ? (
          <>
            <TimerIcon size={22} className="text-muted-foreground" />
            <p className="text-sm text-muted-foreground">{t.noActiveTimer}</p>
            <Button size="sm" onClick={onExpand}><Maximize2 /> {t.expandBtn}</Button>
          </>
        ) : (
          <>
            <div className="flex items-center gap-2">
              <span className={active.timerStartedAt ? 'pulse-dot size-2.5 rounded-full bg-emerald-400 text-emerald-400' : 'size-2.5 rounded-full bg-amber-400'} />
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {active.timerStartedAt ? t.running : t.paused}
              </span>
            </div>
            <h2 className="line-clamp-2 max-w-full text-[15px] font-bold leading-snug">{active.title}</h2>
            <LiveTime task={active} elapsed={active.elapsed} max={active.maxSeconds} className="text-4xl font-black tabular tracking-tight text-emerald-300" showBadge badgeLabel={t.overtime} />
            <div className="mt-1 flex items-center gap-2">
              {active.timerStartedAt ? (
                <Button size="sm" variant="secondary" onClick={onPause}><Pause /> {t.pauseTimer}</Button>
              ) : (
                <Button size="sm" variant="secondary" onClick={onResume}><Play /> {t.resumeTimer}</Button>
              )}
              <Button size="sm" onClick={() => onFinish(active)} className="bg-emerald-600 hover:bg-emerald-500"><Check /> {t.finishTask}</Button>
              <Button size="sm" variant="ghost" onClick={onExpand} title={t.expand}><Maximize2 /></Button>
            </div>
            {onCollapse && (
              <button onClick={onCollapse} title={t.collapse} className="mt-1 flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium text-muted-foreground hover:bg-accent hover:text-foreground">
                <Minimize2 size={12} /> {t.collapse}
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

/** Tiny side-docked horizontal bar: dot + time + title. Click expands to the side view. */
export function TimerWidget({ t, active, onExpand }) {
  return (
    <div
      className="flex h-full cursor-pointer items-center gap-2 bg-background px-2.5"
      style={{ ['--wails-draggable']: 'drag' }}
      onClick={onExpand}
      onDoubleClick={onExpand}
    >
      {!active ? (
        <>
          <TimerIcon size={15} className="shrink-0 text-muted-foreground" />
          <p className="truncate text-[11px] text-muted-foreground">{t.noActiveTimer}</p>
        </>
      ) : (
        <>
          <span className={active.timerStartedAt ? 'pulse-dot size-2 shrink-0 rounded-full bg-emerald-400 text-emerald-400' : 'size-2 shrink-0 rounded-full bg-amber-400'} />
          <LiveTime task={active} elapsed={active.elapsed} max={active.maxSeconds} className="shrink-0 text-[13px] font-black tabular tracking-tight text-emerald-300" />
          <p className="min-w-0 flex-1 truncate text-start text-[11px] font-semibold">{active.title}</p>
          <span style={{ ['--wails-draggable']: 'no-drag' }} className="flex shrink-0 items-center" onClick={(e) => e.stopPropagation()}>
            <button title={t.expand} onClick={onExpand} className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"><ChevronsUp size={14} /></button>
            <button title={t.winHide} onClick={() => WindowHide()} className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"><X size={14} /></button>
          </span>
        </>
      )}
    </div>
  );
}
