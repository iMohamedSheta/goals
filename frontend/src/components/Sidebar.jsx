import { CalendarDays, CalendarRange, Rocket, Star, Settings2, Plus, Database, Power, Languages, Timer, Flag, Target, Compass, Hourglass } from 'lucide-react';
import { cn } from '../lib/utils';
import { Progress } from './ui/form';
import { horizonName, horizonDesc } from '../lib/i18n';

export const HORIZON_ICONS = { short: CalendarDays, medium: CalendarRange, long: Rocket };
const FALLBACK_ICONS = [Flag, Target, Compass, Hourglass];
const fallbackIcon = (key) => {
  let h = 0;
  for (const ch of key) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return FALLBACK_ICONS[h % FALLBACK_ICONS.length];
};

export function Sidebar({
  t, lang, onLangToggle,
  horizons, contexts, counts, focusedCount, stats,
  activeHorizon, onHorizon, focusOnly, onFocusToggle,
  contextFilter, onContextFilter, dbPath, onOpenFolder,
  timerRunning, onQuit, onSettings,
  version, updateAvailable, onOpenUpdates,
}) {
  return (
    <aside className="flex h-full w-64 shrink-0 flex-col border-e bg-card/50">
      <div className="flex items-center gap-2.5 px-5 pb-5 pt-6">
        <div className="flex size-9 items-center justify-center rounded-lg bg-primary shadow-lg shadow-primary/25">
          <Rocket className="text-primary-foreground" size={18} />
        </div>
        <div className="min-w-0 flex-1">
          <h1 className="text-[15px] font-bold leading-none tracking-tight">Goals</h1>
          <p className="mt-1 text-[11px] leading-none text-muted-foreground">{t.tagline}</p>
        </div>
        <button
          onClick={onLangToggle}
          className="flex items-center gap-1 rounded-md border px-2 py-1 text-[11px] font-bold text-muted-foreground transition-colors hover:border-primary/50 hover:text-foreground"
          title="Language / اللغة"
        >
          <Languages size={12} /> {t.langName}
        </button>
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
                      'tabular rounded-full px-1.5 py-0.5 text-[11px] font-semibold',
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
              <span className="tabular rounded-full bg-secondary px-1.5 py-0.5 text-[11px] font-semibold text-muted-foreground">
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
              <button
                key={c.id}
                onClick={() => onContextFilter(c.id === contextFilter ? 'all' : c.id)}
                className={cn(
                  'flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] transition-colors',
                  contextFilter === c.id ? 'bg-accent font-semibold' : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'
                )}
              >
                <span className="size-2.5 shrink-0 rounded-full ring-2 ring-white/10" style={{ background: c.color }} />
                <span className="flex-1 truncate text-start">{c.name}</span>
              </button>
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
