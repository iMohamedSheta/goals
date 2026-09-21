import * as React from 'react';
import {
  Circle, Timer as TimerIcon, AlertCircle, CheckCircle2, Star, Plus, Pencil, Trash2,
  CalendarClock, LayoutGrid, List as ListIcon, SearchX, Minus, X,
  Play, Pause, Check, PictureInPicture2, Maximize2, Minimize2, ChevronsUp, ChevronDown,
  ChevronRight, CornerDownRight, GripVertical,
} from 'lucide-react';
import { cn } from '../lib/utils';
import { Badge } from './ui/badge';
import { Button } from './ui/button';
import { formatHMS, liveElapsed } from '../lib/i18n';
import {
  DEFAULT_APPEARANCE, density, surfClass,
  motionClass, animClass, hoverShadowClass, RADIUS, RADIUS_SM,
} from '../lib/appearance';
import { buildTree, normalizeParentId, descendantIds, subtreeElapsed, subtreeCounts } from '../lib/tree';
import appIcon from '../assets/appicon.png';
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
        <span className={cn('inline-flex shrink-0 items-center border border-red-500/40 bg-red-500/10 px-1.5 py-0.5 text-[10px] font-bold text-red-400', RADIUS_SM)}>
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

export function ContextBadge({ name, color, ap }) {
  if (!name) return <span className="text-xs text-muted-foreground">—</span>;
  return (
    <span className={cn('inline-flex items-center gap-1.5 border border-border bg-secondary font-medium text-secondary-foreground', RADIUS, density(ap).badge, animClass(ap))}>
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
export function TimerControls({ t, task, onStart, onPause, onFinish, ap }) {
  const running = !!task.timerStartedAt;
  const hasTime = running || (task.elapsedSeconds || 0) > 0;
  return (
    <div
      className={cn(
        'flex items-center gap-1.5 border px-2 py-1.5',
        RADIUS_SM,
        animClass(ap),
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

// ---------- Subtask tree card ----------

const TaskCard = React.memo(function TaskCard({
  t, task, context, depth = 0, children = [], collapsed = false,
  onToggleCollapse, onEdit, onDelete, onToggleFocus, onStart, onPause, onFinish,
  onAddSubtask, onCardDrop, isInvalidNest, ap = DEFAULT_APPEARANCE,
}) {
  const [dragging, setDragging] = React.useState(false);
  const [dropPos, setDropPos] = React.useState(null); // 'before' | 'inside' | 'after'
  const due = dueInfo(task, t);
  const d = density(ap);
  const hasKids = (children || []).length > 0 || (task.childCount || 0) > 0;
  const isSub = depth > 0;

  const handleDragOver = (e) => {
    if (!e.dataTransfer?.types?.includes('text/task-id')) return;
    e.preventDefault();
    e.stopPropagation();
    const rect = e.currentTarget.getBoundingClientRect();
    const ratio = (e.clientY - rect.top) / Math.max(1, rect.height);
    let pos = 'inside';
    if (ratio < 0.22) pos = 'before';
    else if (ratio > 0.78) pos = 'after';
    if (pos === 'inside' && isInvalidNest?.(task.id)) {
      setDropPos(null);
      e.dataTransfer.dropEffect = 'none';
      return;
    }
    e.dataTransfer.dropEffect = 'move';
    setDropPos(pos);
  };

  return (
    <div className={cn(isSub && 'ms-4 border-s-2 border-primary/20 ps-2')}>
      <div
        draggable
        onDragStart={(e) => {
          e.dataTransfer.setData('text/task-id', task.id);
          e.dataTransfer.effectAllowed = 'move';
          setDragging(true);
        }}
        onDragEnd={() => { setDragging(false); setDropPos(null); }}
        onDragOver={handleDragOver}
        onDragLeave={(e) => { e.stopPropagation(); setDropPos(null); }}
        onDrop={(e) => {
          e.preventDefault();
          e.stopPropagation();
          const id = e.dataTransfer.getData('text/task-id');
          const pos = dropPos;
          setDropPos(null);
          if (id && pos) onCardDrop?.(id, task.id, pos);
        }}
        onClick={() => onEdit(task)}
        className={cn(
          'tcard group relative h-auto min-h-[150px] w-full cursor-pointer',
          RADIUS,
          surfClass(ap),
          d.card,
          motionClass(ap),
          hoverShadowClass(ap),
          'hover:border-primary/40',
          'shrink-0 grow-0 basis-auto',
          task.focus && 'border-amber-500/40',
          task.timerStartedAt && 'border-emerald-500/40',
          dragging && 'dragging-card',
          dropPos === 'inside' && 'drop-target',
          dropPos === 'before' && 'drop-before',
          dropPos === 'after' && 'drop-after',
          isSub && 'min-h-[130px]',
        )}
      >
        {dropPos === 'before' && <div className="pointer-events-none absolute -top-1.5 start-2 end-2 h-1 rounded-full bg-primary" />}
        {dropPos === 'after' && <div className="pointer-events-none absolute -bottom-1.5 start-2 end-2 h-1 rounded-full bg-primary" />}
        <div className="flex min-w-0 flex-col">
          <div className="flex items-center gap-1.5">
            <span className="cursor-grab text-muted-foreground/60 hover:text-muted-foreground" title={t.dragHint || 'Drag to reorder · drop on a card to nest'}>
              <GripVertical size={13} />
            </span>
            {hasKids ? (
              <button
                onClick={(e) => { e.stopPropagation(); onToggleCollapse?.(task.id); }}
                className="shrink-0 rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
                title={collapsed ? (t.expandSub || 'Show subtasks') : (t.collapseSub || 'Hide subtasks')}
              >
                {collapsed ? <ChevronRight size={14} /> : <ChevronDown size={14} />}
              </button>
            ) : isSub ? (
              <CornerDownRight size={13} className="shrink-0 text-primary/60" />
            ) : null}
            <div className="flex min-w-0 flex-1 items-center gap-1.5">
              <ContextBadge ap={ap} name={context?.name || task.contextName} color={context?.color || task.contextColor || '#6366f1'} />
              {task.focus && (
                <span className={cn('inline-flex shrink-0 items-center gap-1 border border-amber-500/30 bg-amber-500/10 font-semibold text-amber-400', RADIUS, d.badge, animClass(ap))}>
                  <Star size={10} fill="currentColor" /> {t.focusBadge}
                </span>
              )}
              {hasKids && (
                <span className={cn('inline-flex shrink-0 items-center gap-1 border border-primary/30 bg-primary/10 font-semibold text-primary', RADIUS, d.badge)}>
                  {collapsed ? `${task.childCount ?? children.length} ▸` : `${children.filter((c) => c.status === 'done').length}/${children.length || task.childCount || 0}`}
                </span>
              )}
              {isSub && (
                <span className={cn('inline-flex shrink-0 items-center border border-border bg-secondary font-medium text-muted-foreground', RADIUS_SM, d.badge)}>
                  {t.subtask || 'sub'}
                </span>
              )}
            </div>
            <button
              onClick={(e) => { e.stopPropagation(); onToggleFocus(task); }}
              className={cn('shrink-0 rounded p-1', animClass(ap), task.focus ? 'text-amber-400' : 'text-muted-foreground opacity-0 hover:text-amber-400 group-hover:opacity-100')}
              title="Focus"
            >
              <Star size={15} fill={task.focus ? 'currentColor' : 'none'} />
            </button>
          </div>

          <h4 className="mt-2.5 break-words text-sm font-semibold leading-relaxed">{task.title}</h4>
          {task.description && <p className="cdesc mt-1.5 whitespace-pre-wrap break-words text-xs leading-relaxed text-muted-foreground">{task.description}</p>}

          <div className="mt-3 flex flex-wrap items-center gap-1.5">
            <span className={cn('inline-flex items-center border font-bold uppercase tracking-wide', RADIUS_SM, d.badge, animClass(ap), PRIORITY_LABEL)}>
              {t[task.priority] || task.priority}
            </span>
            {due && (
              <span className={cn('inline-flex items-center gap-1 font-medium tabular', RADIUS_SM, 'border', d.badge, animClass(ap), due.cls)}>
                <CalendarClock size={11} /> {due.label}
              </span>
            )}
            {task.timerStartedAt && hasKids && (
              <span className={cn('inline-flex items-center gap-1 border border-emerald-500/30 bg-emerald-500/10 font-medium text-emerald-300', RADIUS_SM, d.badge)}>
                <TimerIcon size={11} /> {t.parentRunning || 'parent tracking'}
              </span>
            )}
          </div>

          <div className="mt-2.5">
            <TimerControls t={t} task={task} onStart={onStart} onPause={onPause} onFinish={onFinish} ap={ap} />
          </div>

          <div className={cn('mt-1 flex items-center gap-1 opacity-0 group-hover:opacity-100', animClass(ap))}>
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
            <button
              onClick={(e) => { e.stopPropagation(); onAddSubtask?.(task); }}
              className="flex items-center gap-1 rounded-md px-1.5 py-1 text-[11px] font-medium text-muted-foreground hover:bg-primary/10 hover:text-primary"
              title={t.addSubtask || 'Add subtask'}
            >
              <Plus size={12} /> {t.subtask || 'sub'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
});

const Column = React.memo(function Column({
  t, status, rootTasks, childrenMap, byId, contextOf,
  onEdit, onDelete, onToggleFocus, onStart, onPause, onFinish,
  onAddSubtask, onCardDrop, onColumnDrop, collapsedSet, onToggleCollapse, ap = DEFAULT_APPEARANCE,
}) {
  const [over, setOver] = React.useState(false);
  const [dragId, setDragId] = React.useState(null);
  const meta = STATUSES.find((s) => s.key === status) || STATUSES[0];
  const Icon = meta.icon;
  const d = density(ap);

  return (
    <div className={cn(surfClass(ap), RADIUS, 'flex h-full max-h-full min-h-0 w-72 shrink-0 grow-0 basis-72 flex-col bg-card/40')}>
      <div className="flex shrink-0 items-center gap-2 px-3.5 pb-2.5 pt-3.5">
        <Icon size={15} className="text-muted-foreground" />
        <span className="text-[13px] font-semibold">{t[status]}</span>
        <span className={cn('tabular ms-auto bg-secondary px-2 py-0.5 text-[11px] font-semibold text-muted-foreground', RADIUS)}>
          {rootTasks.length}
        </span>
      </div>
      <div
        onDragEnter={(e) => { if (e.dataTransfer?.types?.includes('text/task-id')) setOver(true); }}
        onDragOver={(e) => {
          if (!e.dataTransfer?.types?.includes('text/task-id')) return;
          // only treat as column drop when not over a card (cards stopPropagation)
          e.preventDefault();
          setOver(true);
        }}
        onDragLeave={() => setOver(false)}
        onDrop={(e) => {
          e.preventDefault();
          setOver(false);
          const id = e.dataTransfer.getData('text/task-id');
          if (id) onColumnDrop(id, status);
        }}
        className={cn('colbody thin-scroll mx-2 mb-2 flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto overflow-x-hidden', RADIUS_SM, d.colBody, animClass(ap), over && 'drop-target')}
      >
        {rootTasks.map((task) => (
          <TaskCardWithKids
            key={task.id}
            t={t} task={task} depth={0}
            childrenMap={childrenMap} byId={byId} contextOf={contextOf}
            context={contextOf?.(task.contextId)}
            collapsedSet={collapsedSet} onToggleCollapse={onToggleCollapse}
            onEdit={onEdit} onDelete={onDelete} onToggleFocus={onToggleFocus}
            onStart={onStart} onPause={onPause} onFinish={onFinish}
            onAddSubtask={onAddSubtask} onCardDrop={onCardDrop}
            dragId={dragId} setDragId={setDragId} ap={ap}
          />
        ))}
        {rootTasks.length === 0 && (
          <div className="shrink-0 rounded-lg border border-dashed border-border px-3 py-6 text-center text-xs text-muted-foreground">
            {t.dropHere}
            <div className="mt-1 opacity-70">{t.dropNestHint || ''}</div>
          </div>
        )}
      </div>
    </div>
  );
});

function TaskCardWithKids(props) {
  const { task, childrenMap, collapsedSet, onToggleCollapse, dragId, setDragId, contextOf, context, ...rest } = props;
  const kids = childrenMap.get(task.id) || [];
  const collapsed = collapsedSet?.has(task.id);
  const ctx = context ?? contextOf?.(task.contextId);
  return (
    <div
      onDragStartCapture={() => setDragId?.(task.id)}
      onDragEndCapture={() => setDragId?.(null)}
    >
      <TaskCard
        {...rest}
        task={task}
        context={ctx}
        children={kids}
        collapsed={collapsed}
        onToggleCollapse={onToggleCollapse}
        isInvalidNest={(targetId) => {
          const cur = dragId || null;
          if (!cur) return false;
          if (cur === targetId) return true;
          // use live map for cycle check
          return descendantIds(cur, childrenMap).includes(targetId);
        }}
      />
      {!collapsed && kids.map((child) => (
        <TaskCardWithKids
          key={child.id}
          {...props}
          task={child}
          depth={(props.depth || 0) + 1}
        />
      ))}
    </div>
  );
}
// Shallow cards + indented sibling recursion (see TaskCardWithKids above).

export function KanbanBoard(props) {
  const { t, tasks, onCreateFirst, ap = DEFAULT_APPEARANCE } = props;
  const { roots, childrenMap, byId } = React.useMemo(() => buildTree(tasks || []), [tasks]);
  const [collapsedSet, setCollapsedSet] = React.useState(() => new Set());
  const toggleCollapse = React.useCallback((id) => {
    setCollapsedSet((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const handleCardDrop = React.useCallback((dragId, targetId, position) => {
    if (dragId === targetId) return;
    const drag = byId.get(dragId);
    const target = byId.get(targetId);
    if (!drag || !target) return;
    // cycle guard: cannot nest inside own descendant
    if (position === 'inside' && descendantIds(dragId, childrenMap).includes(targetId)) return;
    if (position === 'inside') {
      props.onTaskMove?.(dragId, { parentId: targetId });
      return;
    }
    // before / after: adopt target's parent; top-level moves also adopt column status
    const targetParent = normalizeParentId(target);
    const targetIsTop = !targetParent;
    let siblings;
    if (targetIsTop) {
      siblings = roots.filter((x) => x.status === target.status && x.id !== dragId);
    } else {
      siblings = [...(childrenMap.get(targetParent) || [])].filter((x) => x.id !== dragId);
    }
    const idx = siblings.findIndex((x) => x.id === targetId);
    const insertAt = position === 'before' ? idx : idx + 1;
    const next = [...siblings];
    // drag may already be represented as full object elsewhere; use drag object
    next.splice(insertAt < 0 ? next.length : insertAt, 0, drag);
    const orderedIds = next.map((x) => x.id);
    const patch = { orderedIds };
    if ((normalizeParentId(drag) || null) !== (targetParent || null)) patch.parentId = targetParent; // null = top-level
    if (targetIsTop && drag.status !== target.status) patch.status = target.status;
    // nested before/after keeps the drag's own status (children shown under parent regardless)
    props.onTaskMove?.(dragId, patch);
  }, [byId, childrenMap, roots, props]);

  const handleColumnDrop = React.useCallback((dragId, status) => {
    const drag = byId.get(dragId);
    if (!drag) return;
    // Dropping on empty column area: un-nest to top-level + move status.
    if (!normalizeParentId(drag) && drag.status === status) {
      // pure reorder-to-end of same column
      const siblings = roots.filter((x) => x.status === status && x.id !== dragId).map((x) => x.id);
      props.onTaskMove?.(dragId, { orderedIds: [...siblings, dragId] });
      return;
    }
    const siblings = roots.filter((x) => x.status === status && x.id !== dragId).map((x) => x.id);
    props.onTaskMove?.(dragId, { parentId: null, status, orderedIds: [...siblings, dragId] });
  }, [byId, roots, props]);

  if ((tasks || []).length === 0) {
    return (
      <div className={cn('flex flex-1 flex-col items-center justify-center gap-3 py-20 text-center animate-slide-in', animClass(ap))}>
        <img src={appIcon} alt="Goals" className="size-14 shrink-0 rounded-2xl border bg-card shadow-sm" />
        <div>
          <h3 className="text-base font-semibold">{t.noMatch}</h3>
          <p className="mt-1 text-sm text-muted-foreground">{t.noMatchSub}</p>
        </div>
        <Button onClick={onCreateFirst}><Plus /> {t.newTask}</Button>
      </div>
    );
  }
  const ctxOf = props.contextOf;
  return (
    <div className="flex h-full min-h-0 flex-1 items-stretch gap-3 overflow-x-auto overflow-y-hidden pb-2">
      {STATUSES.map((s) => (
        <Column
          key={s.key} {...props} t={t} status={s.key}
          rootTasks={roots.filter((x) => x.status === s.key)}
          childrenMap={childrenMap} byId={byId} contextOf={ctxOf}
          collapsedSet={collapsedSet} onToggleCollapse={toggleCollapse}
          onCardDrop={handleCardDrop} onColumnDrop={handleColumnDrop}
          ap={ap}
        />
      ))}
    </div>
  );
}

export function TaskTable({ t, tasks, contextOf, onEdit, onDelete, onToggleFocus, onStart, onPause, onFinish, onTaskMove, onAddSubtask, ap = DEFAULT_APPEARANCE }) {
  const d = density(ap);
  const { roots, childrenMap } = React.useMemo(() => buildTree(tasks || []), [tasks]);
  const [collapsedSet, setCollapsedSet] = React.useState(() => new Set());
  const [dropState, setDropState] = React.useState(null); // {id, pos}
  const toggleCollapse = (id) => setCollapsedSet((prev) => {
    const next = new Set(prev);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    return next;
  });
  const flat = React.useMemo(() => {
    const out = [];
    const walk = (nodes, depth) => {
      for (const n of nodes) {
        out.push({ task: n, depth });
        if (!collapsedSet.has(n.id)) walk(childrenMap.get(n.id) || [], depth + 1);
      }
    };
    walk(roots, 0);
    return out;
  }, [roots, childrenMap, collapsedSet]);

  if ((tasks || []).length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 py-20 text-center">
        <div className={cn(surfClass(ap), RADIUS, 'flex size-14 items-center justify-center border bg-card')}>
          <SearchX size={24} className="text-muted-foreground" />
        </div>
        <h3 className="text-base font-semibold">{t.nothingHere}</h3>
        <p className="text-sm text-muted-foreground">{t.nothingSub}</p>
      </div>
    );
  }
  return (
    <div className={cn(surfClass(ap), RADIUS, 'overflow-hidden')}>
      <div className={cn('grid grid-cols-[28px_minmax(0,1fr)_110px_80px_120px_110px_90px_60px] items-center gap-2 border-b bg-muted/40 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground', d.cell)}>
        <span />
        <span>{t.task}</span><span>{t.context}</span><span>{t.priority}</span><span>{t.totalTime}</span><span>{t.due}</span><span>{t.status}</span><span />
      </div>
      {flat.map(({ task, depth }) => {
        const ctx = contextOf(task.contextId);
        const due = dueInfo(task, t);
        const sm = STATUSES.find((s) => s.key === task.status) || STATUSES[0];
        const SIcon = sm.icon;
        const running = !!task.timerStartedAt;
        const kids = childrenMap.get(task.id) || [];
        const ds = dropState?.id === task.id ? dropState.pos : null;
        return (
          <div
            key={task.id}
            draggable
            onDragStart={(e) => { e.dataTransfer.setData('text/task-id', task.id); e.dataTransfer.effectAllowed = 'move'; }}
            onDragOver={(e) => {
              if (!e.dataTransfer?.types?.includes('text/task-id')) return;
              e.preventDefault(); e.stopPropagation();
              const rect = e.currentTarget.getBoundingClientRect();
              const ratio = (e.clientY - rect.top) / Math.max(1, rect.height);
              setDropState({ id: task.id, pos: ratio < 0.25 ? 'before' : ratio > 0.75 ? 'after' : 'inside' });
            }}
            onDragLeave={() => setDropState((s) => (s?.id === task.id ? null : s))}
            onDrop={(e) => {
              e.preventDefault(); e.stopPropagation();
              const dragId = e.dataTransfer.getData('text/task-id');
              const pos = dropState?.id === task.id ? dropState.pos : 'inside';
              setDropState(null);
              if (!dragId || dragId === task.id) return;
              if (pos === 'inside') {
                onTaskMove?.(dragId, { parentId: task.id });
              } else {
                // reorder within the same parent group (table mixes statuses)
                const parent = normalizeParentId(task);
                const group = parent
                  ? [...(childrenMap.get(parent) || [])].filter((x) => x.id !== dragId)
                  : roots.filter((x) => x.id !== dragId);
                const idx = group.findIndex((x) => x.id === task.id);
                const next = [...group];
                const dragged = (tasks || []).find((x) => x.id === dragId) || { id: dragId };
                next.splice(pos === 'before' ? Math.max(0, idx) : idx + 1, 0, dragged);
                onTaskMove?.(dragId, { parentId: parent, orderedIds: next.map((x) => x.id) });
              }
            }}
            onClick={() => onEdit(task)}
            style={{ marginInlineStart: depth > 0 ? depth * 18 : 0 }}
            className={cn(
              'grid cursor-pointer grid-cols-[28px_minmax(0,1fr)_110px_80px_120px_110px_90px_60px] items-center gap-2 border-b last:border-0 hover:bg-accent/50',
              d.cell, animClass(ap),
              ds === 'inside' && 'bg-primary/10 outline outline-2 outline-dashed outline-primary/60',
            )}
          >
            <span className="flex items-center">
              {kids.length > 0 ? (
                <button onClick={(e) => { e.stopPropagation(); toggleCollapse(task.id); }} className="rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground">
                  {collapsedSet.has(task.id) ? <ChevronRight size={14} /> : <ChevronDown size={14} />}
                </button>
              ) : depth > 0 ? (
                <CornerDownRight size={13} className="text-primary/60" />
              ) : (
                <button
                  onClick={(e) => { e.stopPropagation(); onToggleFocus(task); }}
                  className={cn(task.focus ? 'text-amber-400' : 'text-muted-foreground hover:text-amber-400', animClass(ap))}
                >
                  <Star size={15} fill={task.focus ? 'currentColor' : 'none'} />
                </button>
              )}
            </span>
            <span className="min-w-0">
              <span className={cn('block truncate text-[13.5px] font-semibold', task.status === 'done' && 'text-muted-foreground line-through')}>
                {task.title}
                {kids.length > 0 && <span className="ms-2 text-[11px] font-medium text-primary">· {kids.filter((c) => c.status === 'done').length}/{kids.length}</span>}
              </span>
              {task.description && <span className="cdesc block truncate text-xs text-muted-foreground">{task.description}</span>}
            </span>
            <span className="min-w-0"><ContextBadge ap={ap} name={ctx?.name || task.contextName} color={ctx?.color || task.contextColor || '#6366f1'} /></span>
            <span>
              <span className={cn('inline-flex items-center border font-bold uppercase tracking-wide', RADIUS_SM, d.badge, PRIORITY_LABEL)}>
                {t[task.priority] || task.priority}
              </span>
            </span>
            <span className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
              {running && <span className="pulse-dot size-1.5 shrink-0 rounded-full bg-emerald-400 text-emerald-400" />}
              <LiveTime task={task} className={cn('text-xs font-bold', running ? 'text-emerald-300' : 'text-muted-foreground')} showBadge badgeLabel={t.overtime} />
              {!running ? (
                <button title={t.startTimer} onClick={() => onStart(task)} className={cn('rounded p-1 text-muted-foreground hover:bg-emerald-500/15 hover:text-emerald-300', animClass(ap))}><Play size={13} /></button>
              ) : (
                <button title={t.pauseTimer} onClick={() => onPause(task)} className={cn('rounded p-1 text-emerald-300 hover:bg-emerald-500/20', animClass(ap))}><Pause size={13} /></button>
              )}
              <button title={t.finishTask} onClick={() => onFinish(task)} className={cn('rounded p-1 text-muted-foreground hover:bg-primary/15 hover:text-primary', animClass(ap))}><Check size={13} /></button>
            </span>
            <span className="text-xs tabular">
              {due ? <span className={cn('inline-flex items-center gap-1 border', RADIUS_SM, d.badge, animClass(ap), due.cls)}><CalendarClock size={11} />{due.label}</span> : <span className="text-muted-foreground">—</span>}
            </span>
            <span>
              <Badge variant="secondary" className={d.badge}>
                <SIcon size={12} /> {t[task.status]}
              </Badge>
            </span>
            <span className="flex justify-end gap-0.5">
              <button onClick={(e) => { e.stopPropagation(); onEdit(task); }} className={cn('rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground', animClass(ap))}><Pencil size={14} /></button>
              <button onClick={(e) => { e.stopPropagation(); onDelete(task); }} className={cn('rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-red-400', animClass(ap))}><Trash2 size={14} /></button>
            </span>
          </div>
        );
      })}
    </div>
  );
}

export function ViewToggle({ t, view, onChange, ap }) {
  return (
    <div className={cn('flex border bg-secondary/60 p-0.5', RADIUS_SM, animClass(ap))}>
      {[
        { key: 'kanban', label: t.board, icon: LayoutGrid },
        { key: 'list', label: t.list, icon: ListIcon },
      ].map((v) => (
        <button
          key={v.key}
          onClick={() => onChange(v.key)}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 text-[13px] font-medium',
            RADIUS_SM,
            animClass(ap),
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
export function ActiveTimerPill({ t, active, activeCount, onPause, onResume, onMini, onFinish, ap }) {
  if (!active) return null;
  const running = !!active.timerStartedAt;
  return (
    <div className={cn('flex items-center gap-2 border border-emerald-500/40 bg-emerald-500/10 py-1 pe-1 ps-3 animate-slide-in', RADIUS, animClass(ap))}>
      <span className={running ? 'pulse-dot size-2 rounded-full bg-emerald-400 text-emerald-400' : 'size-2 rounded-full bg-amber-400'} />
      <span className="max-w-[180px] truncate text-[13px] font-semibold text-emerald-200">{active.title}</span>
      {(activeCount || 1) > 1 && (
        <span className="rounded-full bg-emerald-500/25 px-2 py-0.5 text-[11px] font-bold text-emerald-100" title={t.multiRunning || 'Parent + subtask running'}>
          +{(activeCount || 1) - 1}
        </span>
      )}
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

/** Header pill for the running context goal — sits next to the task pill; both may run. */
export function ActiveContextPill({ t, active, onPause, onResume, ap }) {
  if (!active) return null;
  const running = !!active.timerStartedAt;
  const weekly = (active.recurrence || 'daily') === 'weekly';
  const periodValue = weekly ? (active.weekSeconds || 0) : (active.todaySeconds || 0);
  const pct = active.dailyTargetSeconds > 0
    ? Math.min(100, Math.round((periodValue / active.dailyTargetSeconds) * 100))
    : 0;
  return (
    <div className={cn('flex items-center gap-2 border border-violet-500/40 bg-violet-500/10 py-1 pe-1 ps-3 animate-slide-in', RADIUS, animClass(ap))}>
      <span className="size-2 shrink-0 rounded-full ring-2 ring-white/10" style={{ background: active.color || '#8b5cf6' }} />
      <span className={running ? 'pulse-dot size-2 rounded-full bg-violet-400 text-violet-400' : 'size-2 rounded-full bg-amber-400'} />
      <span className="max-w-[180px] truncate text-[13px] font-semibold text-violet-200">{active.name}</span>
      <LiveTime task={active} elapsed={active.elapsed} max={active.maxSeconds} className="text-[13px] font-bold tabular text-violet-300" showBadge badgeLabel={t.overtime} />
      {active.dailyTargetSeconds > 0 && (
        <span className="tabular hidden text-[11px] text-violet-300/80 xl:inline" title={weekly ? t.weeklyTarget : t.dailyTarget}>
          {formatHMS(periodValue)}/{formatHMS(active.dailyTargetSeconds)} · {pct}%
        </span>
      )}
      {running ? (
        <button title={t.pauseTimer} onClick={onPause} className="rounded-md p-1.5 text-violet-300 hover:bg-violet-500/20"><Pause size={14} /></button>
      ) : (
        <button title={t.resumeTimer} onClick={onResume} className="rounded-md p-1.5 text-violet-300 hover:bg-violet-500/20"><Play size={14} /></button>
      )}
    </div>
  );
}

/** Full-window mini mode: task timer + context-goal timer. Small + always on top. */
export function MiniTimer({ t, active, onPause, onResume, onFinish, onExpand, onCollapse, ctx, onPauseCtx, onResumeCtx, lastCtx, onResumeLastCtx }) {
  const showCtx = !!ctx;
  const ctxRunning = !!ctx?.timerStartedAt;
  return (
    <div className="flex h-full flex-col bg-background">
      <div className="flex h-8 shrink-0 select-none items-center border-b px-1" style={{ ['--wails-draggable']: 'drag' }}>
        <span className="pointer-events-none px-2 text-[11px] font-bold text-muted-foreground">Goals · {t.activeTimer}</span>
        <span className="ms-auto flex items-stretch" style={{ ['--wails-draggable']: 'no-drag' }}>
          <MiniCap t={t} onExpand={onExpand} />
        </span>
      </div>
      <div className="thin-scroll flex min-h-0 flex-1 flex-col items-center justify-center gap-3 overflow-y-auto px-6 py-3 text-center">
        {!active && !showCtx ? (
          <>
            <TimerIcon size={22} className="text-muted-foreground" />
            <p className="text-sm text-muted-foreground">{t.noActiveTimer}</p>
            <Button size="sm" onClick={onExpand}><Maximize2 /> {t.expandBtn}</Button>
          </>
        ) : (
          <>
            {active && (
              <div className="flex flex-col items-center gap-2">
                <div className="flex items-center gap-2">
                  <span className={active.timerStartedAt ? 'pulse-dot size-2.5 rounded-full bg-emerald-400 text-emerald-400' : 'size-2.5 rounded-full bg-amber-400'} />
                  <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    {active.timerStartedAt ? t.running : t.paused}
                  </span>
                </div>
                <h2 className="line-clamp-2 max-w-full text-[15px] font-bold leading-snug">{active.title}</h2>
                <LiveTime task={active} elapsed={active.elapsed} max={active.maxSeconds} className="text-4xl font-black tabular tracking-tight text-emerald-300" showBadge badgeLabel={t.overtime} />
                {active.maxSeconds > 0 && (
                  <span className="tabular text-[11px] text-muted-foreground">{t.maxTime}: {formatHMS(active.maxSeconds)}</span>
                )}
                <div className="mt-1 flex items-center gap-2">
                  {active.timerStartedAt ? (
                    <Button size="sm" variant="secondary" onClick={onPause}><Pause /> {t.pauseTimer}</Button>
                  ) : (
                    <Button size="sm" variant="secondary" onClick={onResume}><Play /> {t.resumeTimer}</Button>
                  )}
                  <Button size="sm" onClick={() => onFinish(active)} className="bg-emerald-600 hover:bg-emerald-500"><Check /> {t.finishTask}</Button>
                  <Button size="sm" variant="ghost" onClick={onExpand} title={t.expand}><Maximize2 /></Button>
                </div>
              </div>
            )}
            {showCtx && (
              <div className={active ? 'flex w-full flex-col items-center gap-1.5 border-t border-violet-500/20 pt-3' : 'flex flex-col items-center gap-2'}>
                <div className="flex items-center gap-2">
                  <span className="size-2.5 shrink-0 rounded-full ring-2 ring-white/10" style={{ background: ctx.color || '#8b5cf6' }} />
                  <span className={ctxRunning ? 'pulse-dot size-2.5 rounded-full bg-violet-400 text-violet-400' : 'size-2.5 rounded-full bg-amber-400'} />
                  <span className="max-w-[200px] truncate text-[13px] font-bold">{ctx.name}</span>
                </div>
                <LiveTime
                  task={ctx}
                  elapsed={ctx.elapsed}
                  max={ctx.maxSeconds}
                  className={active ? 'text-2xl font-black tabular tracking-tight text-violet-300' : 'text-4xl font-black tabular tracking-tight text-violet-300'}
                  showBadge
                  badgeLabel={t.overtime}
                />
                {ctx.maxSeconds > 0 && (
                  <span className="tabular text-[11px] text-muted-foreground">{t.maxTime}: {formatHMS(ctx.maxSeconds)}</span>
                )}
                <div className="mt-0.5 flex items-center gap-2">
                  {ctxRunning ? (
                    <Button size="sm" variant="secondary" onClick={onPauseCtx}><Pause /> {t.pauseTimer}</Button>
                  ) : (
                    <Button size="sm" variant="secondary" onClick={onResumeCtx}><Play /> {t.resumeTimer}</Button>
                  )}
                  {!active && (
                    <Button size="sm" variant="ghost" onClick={onExpand} title={t.expand}><Maximize2 /></Button>
                  )}
                </div>
              </div>
            )}
            {!showCtx && lastCtx && (
              <div className="flex flex-col items-center gap-1.5 rounded-lg border border-dashed border-violet-500/30 px-5 py-2.5">
                <span className="max-w-[200px] truncate text-xs font-semibold text-violet-200/80">
                  {lastCtx.name} · {t.paused}
                </span>
                <Button size="sm" variant="secondary" onClick={onResumeLastCtx}><Play /> {t.resumeTimer}</Button>
              </div>
            )}
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

/** Side-docked mini bar: task and/or context timer rows. Click expands to the side view. */
export function TimerWidget({ t, active, ctx, lastCtx, onExpand }) {
  const taskRow = active && {
    timerStartedAt: active.timerStartedAt,
    elapsed: active.elapsed,
    maxSeconds: active.maxSeconds,
    title: active.title,
    violet: false,
  };
  const ctxRow = ctx && {
    timerStartedAt: ctx.timerStartedAt,
    elapsedSeconds: ctx.elapsedSeconds ?? ctx.elapsed,
    elapsed: ctx.elapsed,
    maxSeconds: ctx.maxSeconds,
    title: ctx.name,
    violet: true,
  };
  const rows = [taskRow, ctxRow].filter(Boolean);
  return (
    <div
      className="flex h-full cursor-pointer items-center gap-2 bg-background px-2.5"
      style={{ ['--wails-draggable']: 'drag' }}
      onClick={onExpand}
      onDoubleClick={onExpand}
    >
      {rows.length === 0 ? (
        <>
          <TimerIcon size={15} className="shrink-0 text-muted-foreground" />
          <p className="min-w-0 flex-1 truncate text-[11px] text-muted-foreground">
            {lastCtx ? `${lastCtx.name} · ${t.paused}` : t.noActiveTimer}
          </p>
          <span style={{ ['--wails-draggable']: 'no-drag' }} className="flex shrink-0 items-center" onClick={(e) => e.stopPropagation()}>
            <button title={t.expand} onClick={onExpand} className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"><ChevronsUp size={14} /></button>
            <button title={t.winHide} onClick={() => WindowHide()} className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"><X size={14} /></button>
          </span>
        </>
      ) : (
        <>
          <div className="flex min-w-0 flex-1 flex-col justify-center gap-1">
            {rows.map((r, i) => (
              <div key={i} className="flex min-w-0 items-center gap-1.5">
                <span className={r.timerStartedAt ? `pulse-dot size-1.5 shrink-0 rounded-full ${r.violet ? 'bg-violet-400 text-violet-400' : 'bg-emerald-400 text-emerald-400'}` : 'size-1.5 shrink-0 rounded-full bg-amber-400'} />
                <LiveTime task={r} elapsed={r.elapsed} max={r.maxSeconds} className={`shrink-0 text-[12px] font-black tabular tracking-tight ${r.violet ? 'text-violet-300' : 'text-emerald-300'}`} />
                <p className="min-w-0 flex-1 truncate text-start text-[10px] font-semibold">{r.title}</p>
              </div>
            ))}
          </div>
          <span style={{ ['--wails-draggable']: 'no-drag' }} className="flex shrink-0 items-center" onClick={(e) => e.stopPropagation()}>
            <button title={t.expand} onClick={onExpand} className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"><ChevronsUp size={14} /></button>
            <button title={t.winHide} onClick={() => WindowHide()} className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"><X size={14} /></button>
          </span>
        </>
      )}
    </div>
  );
}
