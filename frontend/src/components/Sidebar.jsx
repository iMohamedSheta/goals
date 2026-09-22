import * as React from 'react';
import { CalendarDays, CalendarRange, Rocket, Star, Settings2, Plus, Database, Power, Languages, Timer, Flag, Target, Compass, Hourglass, Play, Pause } from 'lucide-react';
import { cn } from '../lib/utils';
import { Progress } from './ui/form';
import { horizonName, horizonDesc, formatHMS } from '../lib/i18n';
import { LiveTime } from './Board';
import { RADIUS, animClass, DEFAULT_APPEARANCE } from '../lib/appearance';
import sideLogoDark from '../assets/side-logo-dark.png';
import sideLogoLight from '../assets/side-logo-light.png';

export const HORIZON_ICONS = { short: CalendarDays, medium: CalendarRange, long: Rocket };
const FALLBACK_ICONS = [Flag, Target, Compass, Hourglass];
const fallbackIcon = (key) => {
  let h = 0;
  for (const ch of key) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return FALLBACK_ICONS[h % FALLBACK_ICONS.length];
};

function ContextGoalRow({ t, lang, c, selected, onFilter, onStart, onPause }) {
  const running = !!c.timerStartedAt;
  const target = c.dailyTargetSeconds || 0;
  const weekly = (c.recurrence || 'daily') === 'weekly';
  const periodBase = weekly ? (c.weekSeconds || 0) : (c.todaySeconds || 0);
  const periodLabel = weekly ? t.weekLabel : t.todayLabel;
  const periodDoneLabel = weekly ? t.weekDone : t.dailyDone;
  const totalOwn = c.totalSeconds ?? c.elapsedSeconds ?? 0;
  const todayOwn = c.todaySeconds || 0;
  const desc = lang === 'ar' ? (c.descriptionAr || c.description) : (c.description || c.descriptionAr);
  // Period progress ticks live while running: anchor on the last fetched value
  // and add only the seconds elapsed since that fetch (avoids double counting
  // the live delta the backend already included in the period value).
  const [now, setNow] = React.useState(Date.now());
  const [anchor, setAnchor] = React.useState({ value: periodBase, at: Date.now() });
  React.useEffect(() => {
    setAnchor({ value: periodBase, at: Date.now() });
  }, [periodBase, c.timerStartedAt]); // eslint-disable-line
  React.useEffect(() => {
    if (!running) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [running, c.timerStartedAt]);
  const shown = anchor.value + (running ? Math.max(0, Math.floor((now - anchor.at) / 1000)) : 0);
  const pct = target > 0 ? Math.min(100, Math.round((shown / target) * 100)) : 0;
  const done = target > 0 && shown >= target;
  const showTotal = running || totalOwn > 0;
  const stop = (e) => { e.stopPropagation(); };
  const totalTitle = `${t.todayTime || t.todayLabel}: ${formatHMS(todayOwn)} · ${t.totalLabel || 'Total'}: ${formatHMS(totalOwn)}`;
  return (
    // Whole card is the filter target — click anywhere except the timer button.
    <div
      onClick={onFilter}
      title={desc || c.name}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onFilter?.(); } }}
      className={cn(
        'cursor-pointer overflow-hidden rounded-lg border transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        selected ? 'border-primary/40 bg-accent' : 'border-transparent hover:bg-accent/60',
        running && 'border-violet-500/30 bg-violet-500/5'
      )}
    >
      <div className="flex items-center gap-2.5 px-3 pt-2.5">
        <span className="size-3 shrink-0 rounded-full ring-2 ring-white/10" style={{ background: c.color }} />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-semibold leading-tight">{c.name}</span>
          {desc ? (
            <span className="mt-0.5 block truncate text-[11px] leading-tight text-muted-foreground">{desc}</span>
          ) : null}
          {target > 0 ? (
            <span className="tabular mt-1 block truncate text-[11px] leading-none text-muted-foreground">
              {periodLabel}: {formatHMS(shown)}/{formatHMS(target)} ·{' '}
              <span className={cn('font-bold', done ? 'text-emerald-400' : 'text-violet-300/90')}>{done ? periodDoneLabel : `${pct}%`}</span>
            </span>
          ) : showTotal ? (
            <span className="tabular mt-1 block truncate text-[11px] leading-none text-muted-foreground" title={totalTitle}>
              {t.todayTime || t.todayLabel}: {formatHMS(todayOwn)} · {t.totalLabel || 'Total'}:{' '}
              <LiveTime task={c} className={cn('font-bold', running ? 'text-violet-300' : 'text-muted-foreground')} />
            </span>
          ) : null}
        </span>
        <button
          title={running ? t.pauseTimer : t.startTimer}
          onClick={(e) => { stop(e); (running ? onPause : onStart)?.(); }}
          className={cn(
            'grid size-9 shrink-0 place-items-center rounded-lg border transition-colors',
            running
              ? 'border-violet-500/40 bg-violet-500/15 text-violet-300 hover:bg-violet-500/25'
              : 'border-border bg-secondary/60 text-muted-foreground hover:border-violet-500/40 hover:text-violet-300'
          )}
        >
          {running ? <Pause size={15} /> : <Play size={15} />}
        </button>
      </div>
      {target > 0 && (
        <div className="px-3 pb-2.5 pt-1.5">
          <Progress value={pct} />
          {showTotal && (
            <p className="tabular mt-1.5 text-[11px] leading-none text-muted-foreground" title={totalTitle}>
              {t.totalLabel || 'Total'}: {formatHMS(totalOwn)}
            </p>
          )}
        </div>
      )}
    </div>
  );
}

export function Sidebar({
  t, lang, onLangToggle,
  horizons, contexts, counts, focusedCount, stats,
  activeHorizon, onHorizon, focusOnly, onFocusToggle,
  contextFilter, onContextFilter, dbPath, onOpenFolder,
  timerRunning, onQuit, onSettings,
  version, updateAvailable, onOpenUpdates,
  ap = DEFAULT_APPEARANCE,
  onStartContext, onPauseContext,
}) {
  return (
    <aside className="flex h-full w-64 shrink-0 flex-col border-e bg-card/50">
      <div className="px-5 pb-4 pt-6">
        <div className="flex items-center gap-2">
          <img src={sideLogoLight} alt="Goals" className="h-10 w-auto min-w-0 flex-1 object-contain object-left dark:hidden" />
          <img src={sideLogoDark} alt="Goals" className="hidden h-10 w-auto min-w-0 flex-1 object-contain object-left dark:block" />
          <button
            onClick={onLangToggle}
            className="flex shrink-0 items-center gap-1 rounded-md border px-2 py-1 text-[11px] font-bold text-muted-foreground transition-colors hover:border-primary/50 hover:text-foreground"
            title="Language / اللغة"
          >
            <Languages size={12} /> {t.langName}
          </button>
        </div>
        <p className="mt-2 text-[11px] leading-none text-muted-foreground">{t.tagline}</p>
      </div>

      <div className="flex-1 space-y-6 overflow-y-auto px-3">
        <div>
          <p className="px-2 pb-2 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{t.planning}</p>
          <div className="space-y-1">
            {horizons.map((hz) => {
              const Icon = HORIZON_ICONS[hz.key] || fallbackIcon(hz.key);
              const c = counts[hz.key] || { total: 0, done: 0 };
              const pct = c.total ? Math.round((c.done / c.total) * 100) : 0;
              const active = activeHorizon === hz.key && !focusOnly;
              return (
                <button
                  key={hz.key}
                  onClick={() => onHorizon(hz.key)}
                  title={horizonDesc(hz, lang)}
                  className={cn(
                    'group w-full rounded-lg px-2.5 py-2 text-start transition-colors',
                    active ? 'bg-accent' : 'hover:bg-accent/60'
                  )}
                >
                  <div className="flex items-center gap-2.5">
                    <span className={cn(
                      'flex size-7 items-center justify-center rounded-md border transition-colors',
                      active ? 'border-primary/40 bg-primary/15 text-primary' : 'border-border bg-secondary text-muted-foreground group-hover:text-foreground'
                    )}>
                      <Icon size={15} />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-[13px] font-semibold leading-none">{horizonName(hz, lang)}</span>
                      <span className="tabular mt-1 block text-[11px] leading-none text-muted-foreground">{t.horizonSub(hz.defaultDays, c.done, c.total)}</span>
                    </span>
                    <span className={cn(
                      'tabular px-1.5 py-0.5 text-[11px] font-semibold',
                      RADIUS,
                      animClass(ap),
                      active ? 'bg-primary/15 text-primary' : 'bg-secondary text-muted-foreground'
                    )}>
                      {c.total}
                    </span>
                  </div>
                  <Progress value={pct} className="mt-2 h-1" />
                </button>
              );
            })}
            <button
              onClick={onFocusToggle}
              className={cn(
                'flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-start transition-colors',
                focusOnly ? 'bg-amber-500/10 ring-1 ring-inset ring-amber-500/40' : 'hover:bg-accent/60'
              )}
            >
              <span className={cn(
                'flex size-7 items-center justify-center rounded-md border',
                focusOnly ? 'border-amber-500/40 bg-amber-500/15 text-amber-400' : 'border-border bg-secondary text-muted-foreground'
              )}>
                <Star size={15} fill={focusOnly ? 'currentColor' : 'none'} />
              </span>
              <span className="flex-1 text-[13px] font-semibold">{t.focus}</span>
              <span className={cn('tabular bg-secondary px-1.5 py-0.5 text-[11px] font-semibold text-muted-foreground', RADIUS, animClass(ap))}>
                {focusedCount}
              </span>
            </button>
          </div>
        </div>

        <div>
          <div className="flex items-center justify-between px-2 pb-2">
            <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">{t.contexts}</p>
            <button onClick={() => onSettings('contexts')} className="rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground" title={t.manageContexts}>
              <Plus size={14} />
            </button>
          </div>
          <div className="space-y-0.5">
            <button
              onClick={() => onContextFilter('all')}
              className={cn(
                'flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] transition-colors',
                contextFilter === 'all' ? 'bg-accent font-semibold' : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'
              )}
            >
              <span className="size-2.5 rounded-full bg-muted-foreground/60" />
              <span className="flex-1 text-start">{t.allContexts}</span>
            </button>
            {contexts.map((c) => (
              <ContextGoalRow
                key={c.id}
                t={t}
                lang={lang}
                c={c}
                selected={contextFilter === c.id}
                onFilter={() => onContextFilter(c.id === contextFilter ? 'all' : c.id)}
                onStart={() => onStartContext?.(c)}
                onPause={() => onPauseContext?.(c)}
              />
            ))}
          </div>
        </div>

        {timerRunning && (
          <div className="flex items-start gap-2 rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-2.5 text-[11px] leading-relaxed text-emerald-300/90">
            <Timer size={14} className="mt-0.5 shrink-0" />
            <span>{t.bgHint}</span>
          </div>
        )}
      </div>

      <div className="space-y-1 border-t p-3">
        <button
          onClick={onSettings}
          className="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-[13px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <Settings2 size={15} /> {t.settings}
        </button>
        <button
          onClick={onQuit}
          title={t.quitMsg}
          className="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-[13px] font-medium text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-red-400"
        >
          <Power size={15} /> {t.quit}
        </button>
        <button
          onClick={() => onOpenFolder?.()}
          title={`${t.openFolder}\n${dbPath || ''}`}
          className="flex w-full items-center gap-2 rounded-lg px-2.5 py-1.5 text-start text-[11px] text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <Database size={12} className="shrink-0" />
          <span className="min-w-0 flex-1 truncate" dir="ltr">{dbPath ? dbPath.split(/[/\\]/).slice(-2).join('/') : '…'}</span>
          {stats && <span className="tabular shrink-0">{stats.done}/{stats.total} ✓</span>}
        </button>
        <button
          onClick={() => onOpenUpdates?.()}
          title={t.openUpdates}
          className="flex w-full items-center gap-2 rounded-lg px-2.5 py-1.5 text-start text-[11px] text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <span dir="ltr" className="tabular shrink-0">v{version || 'dev'}</span>
          {updateAvailable && (
            <span className="flex min-w-0 items-center gap-1.5 font-bold text-emerald-400">
              <span className="size-1.5 shrink-0 animate-pulse rounded-full bg-emerald-400" />
              <span className="truncate">{t.updateAvailableShort}</span>
            </span>
          )}
        </button>
      </div>
    </aside>
  );
}
