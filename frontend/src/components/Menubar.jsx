import * as React from 'react';
import {
  Plus, Star, X, Minus, Square, LayoutGrid, List as ListIcon, CalendarDays, CalendarRange,
  Rocket, Settings2, Users, Languages, Power, Check, PictureInPicture2, Palette,
} from 'lucide-react';
import { cn } from '../lib/utils';
import { horizonName } from '../lib/i18n';
import { WindowHide, WindowMinimise, WindowToggleMaximise } from '../../wailsjs/runtime/runtime';

export const DRAG = { ['--wails-draggable']: 'drag' };
export const NODRAG = { ['--wails-draggable']: 'no-drag' };

/** Normal OS-style caption buttons (min / max / close-to-tray). */
export function WindowControls({ t, small }) {
  const base = cn(
    'flex items-center justify-center text-muted-foreground transition-colors',
    small ? 'h-7 w-10' : 'h-9 w-12'
  );
  return (
    <div className="flex items-stretch" style={NODRAG}>
      <button title={t.winMin} onClick={() => WindowMinimise()} className={cn(base, 'hover:bg-accent hover:text-foreground')}>
        <Minus size={15} />
      </button>
      <button title={t.winMax} onClick={() => WindowToggleMaximise()} className={cn(base, 'hover:bg-accent hover:text-foreground')}>
        <Square size={12} />
      </button>
      <button title={t.winHide} onClick={() => WindowHide()} className={cn(base, 'hover:bg-[#E81123] hover:text-white')}>
        <X size={16} />
      </button>
    </div>
  );
}

const HICONS = { short: CalendarDays, medium: CalendarRange, long: Rocket };

function Item({ icon: Icon, label, shortcut, checked, danger, disabled, onClick }) {
  return (
    <button
      disabled={disabled}
      onClick={onClick}
      className={cn(
        'flex w-full items-center gap-2.5 rounded-md px-2.5 py-[7px] text-[13px] transition-colors',
        danger
          ? 'text-red-400 hover:bg-destructive/10'
          : 'text-foreground/90 hover:bg-accent hover:text-foreground',
        disabled && 'pointer-events-none opacity-40'
      )}
    >
      <span className="flex w-4 justify-center text-muted-foreground">
        {checked ? <Check size={14} className="text-primary" /> : Icon ? <Icon size={14} /> : null}
      </span>
      <span className="flex-1 text-start font-medium">{label}</span>
      {shortcut && (
        <kbd className="rounded border bg-muted px-1.5 py-0.5 text-[10px] font-semibold text-muted-foreground">
          {shortcut}
        </kbd>
      )}
    </button>
  );
}

function Sep() {
  return <div className="mx-2 my-1 h-px bg-border" />;
}

export function Menubar({
  t, lang, horizons, activeHorizon, view, focusOnly, hasFilters, timerRunning,
  onNewTask, onView, onHorizon, onFocusToggle, onClearFilters, onMini,
  onSettings, onLangToggle, onQuit,
}) {
  const [open, setOpen] = React.useState(null);
  const close = () => setOpen(null);

  React.useEffect(() => {
    if (!open) return;
    const onKey = (e) => e.key === 'Escape' && close();
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open ]);

  const run = (fn) => () => { close(); fn?.(); };

  const menus = [
    {
      key: 'tasks',
      label: t.menuTasks,
      items: [
        { icon: Plus, label: t.newTask, shortcut: t.newTaskShortcut, onClick: run(onNewTask) },
        { icon: Star, label: t.focus, checked: focusOnly, onClick: run(onFocusToggle) },
        { type: 'sep' },
        { icon: X, label: t.clear, disabled: !hasFilters, onClick: run(onClearFilters) },
      ],
    },
    {
      key: 'view',
      label: t.menuView,
      items: [
        { icon: LayoutGrid, label: t.board, checked: view === 'kanban', onClick: run(() => onView('kanban')) },
        { icon: ListIcon, label: t.list, checked: view === 'list', onClick: run(() => onView('list')) },
        { type: 'sep' },
        ...horizons.map((h) => ({
          icon: HICONS[h.key] || CalendarDays,
          label: horizonName(h, lang),
          checked: activeHorizon === h.key,
          onClick: run(() => onHorizon(h.key)),
        })),
        { type: 'sep' },
        { icon: PictureInPicture2, label: t.miniMode, disabled: !timerRunning, onClick: run(onMini) },
      ],
    },
    {
      key: 'manage',
      label: t.menuManage,
      items: [
        { icon: Settings2, label: t.timelines, onClick: run(() => onSettings('planning')) },
        { icon: Users, label: t.contexts, onClick: run(() => onSettings('contexts')) },
        { icon: Palette, label: t.appearance, onClick: run(() => onSettings('appearance')) },
        { type: 'sep' },
        { icon: Languages, label: `${t.language}: ${t.langName}`, onClick: run(onLangToggle) },
        { type: 'sep' },
        { icon: Power, label: t.quit, danger: true, onClick: run(onQuit) },
      ],
    },
  ];

  return (
    <div
      className="relative z-30 flex h-9 shrink-0 select-none items-center gap-0.5 border-b bg-card/60 pe-0 ps-2"
      style={DRAG}
      onDoubleClick={() => WindowToggleMaximise()}
    >
      {open && <div className="fixed inset-0 z-40 cursor-default" onClick={close} onContextMenu={close} />}
      <div className="flex items-center gap-0.5" style={NODRAG}>
      {menus.map((m) => (
        <div key={m.key} className="relative" style={NODRAG}>
          <button
            onClick={() => setOpen(open === m.key ? null : m.key)}
            onMouseEnter={() => open && open !== m.key && setOpen(m.key)}
            className={cn(
              'rounded-md px-3 py-1.5 text-[13px] font-medium transition-colors',
              open === m.key ? 'bg-accent text-foreground' : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'
            )}
          >
            {m.label}
          </button>
          {open === m.key && (
            <div className="absolute start-0 top-full z-50 mt-1 w-60 rounded-xl border bg-popover p-1.5 shadow-2xl animate-slide-in" style={NODRAG}>
              {m.items.map((it, i) =>
                it.type === 'sep' ? <Sep key={i} /> : <Item key={i} {...it} />
              )}
            </div>
          )}
        </div>
      ))}
      </div>
      <div className="ms-auto">
        <WindowControls t={t} />
      </div>
    </div>
  );
}
