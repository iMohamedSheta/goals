import * as React from 'react';
import { BarChart3, ChevronLeft, ChevronRight, Clock, Flame, Briefcase, Moon, Shapes, ListVideo, Globe, X, Radio } from 'lucide-react';
import { cn } from '../lib/utils';
import { Button } from './ui/button';
import { Input, Select } from './ui/form';
import { Sheet, SheetHeader, SheetBody, SheetFooter } from './ui/sheet';
import { formatHMS } from '../lib/i18n';
import { RADIUS } from '../lib/appearance';
import { GetActivitySummary, ListActivitySegments, GetActivityStatus } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

function dayStr(d) {
  return d.toISOString().slice(0, 10);
}
function shiftDay(s, n) {
  const d = new Date(s + 'T12:00:00Z');
  d.setUTCDate(d.getUTCDate() + n);
  return dayStr(d);
}

function Bar({ value, max, className }) {
  const pct = max > 0 ? Math.min(100, Math.round((value / max) * 100)) : 0;
  return (
    <div className="h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-muted">
      <div className={cn('h-full rounded-full', className)} style={{ width: `${pct}%` }} />
    </div>
  );
}

function Row({ name, secs, max, sessions, accent, sub, onClick, selected }) {
  const body = (
    <div className="min-w-0 flex-1">
      <div className="flex items-baseline justify-between gap-2">
        <p className="truncate text-[13px] font-semibold" title={name}>{name}</p>
        <span className="tabular shrink-0 text-xs text-muted-foreground">{formatHMS(secs)}</span>
      </div>
      <div className="mt-1 flex items-center gap-2">
        <Bar value={secs} max={max} className={accent} />
        {sessions != null && <span className="tabular shrink-0 text-[10px] text-muted-foreground">×{sessions}</span>}
      </div>
      {sub && <p className="mt-0.5 truncate text-[11px] text-muted-foreground">{sub}</p>}
    </div>
  );
  if (!onClick) {
    return <div className="flex items-center gap-2.5 py-1.5">{body}</div>;
  }
  return (
    <button
      onClick={onClick}
      title={name}
      className={cn(
        'flex w-full items-center gap-2.5 rounded-lg px-1.5 py-1.5 text-start transition-colors hover:bg-accent/60',
        selected && 'bg-accent ring-1 ring-inset ring-primary/40'
      )}
    >
      {body}
    </button>
  );
}

function StatCard({ icon: Icon, label, secs, tone }) {
  return (
    <div className={cn('min-w-0 flex-1 rounded-xl border p-3', RADIUS)}>
      <div className="flex items-center gap-1.5 text-[11px] font-bold text-muted-foreground">
        <Icon size={13} className={tone} />
        <span className="truncate">{label}</span>
      </div>
      <p className="tabular mt-1 truncate text-lg font-black">{formatHMS(secs)}</p>
    </div>
  );
}

function fmtTime(s) {
  if (!s) return '';
  try {
    const d = new Date(s);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch {
    return (s || '').slice(11, 16);
  }
}

const catDot = (c) => {
  if (c === 'work') return 'bg-emerald-400';
  if (c === 'distraction') return 'bg-rose-400';
  if (c === 'idle') return 'bg-muted-foreground/50';
  return 'bg-sky-400';
};

export function ActivityInsights({ t, open, onClose, enabled }) {
  const [range, setRange] = React.useState('day');
  const [refDay, setRefDay] = React.useState(() => dayStr(new Date()));
  const [appFilter, setAppFilter] = React.useState('all');
  const [search, setSearch] = React.useState('');
  const [summary, setSummary] = React.useState(null);
  const [segments, setSegments] = React.useState([]);
  const [live, setLive] = React.useState(null);
  const [loading, setLoading] = React.useState(false);

  const load = React.useCallback(async (r, d, app) => {
    setLoading(true);
    try {
      const [s, list] = await Promise.all([
        GetActivitySummary(r, d),
        ListActivitySegments(r, d, app && app !== 'all' ? app : '', 200),
      ]);
      setSummary(s || null);
      setSegments(list || []);
    } catch {
      setSummary(null);
      setSegments([]);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    if (!open) return;
    load(range, refDay, appFilter);
  }, [open, range, refDay, appFilter, load]);

  // live "now" pill + refresh today's totals while open
  React.useEffect(() => {
    if (!open) return;
    GetActivityStatus().then((s) => setLive(s || null)).catch(() => {});
    const off = EventsOn('activity:tick', (s) => {
      setLive(s || null);
      if (range === 'day' && refDay === dayStr(new Date())) {
        GetActivitySummary('day', refDay).then((x) => setSummary(x || null)).catch(() => {});
        ListActivitySegments('day', refDay, appFilter !== 'all' ? appFilter : '', 200).then((x) => setSegments(x || [])).catch(() => {});
      }
    });
    const id = setInterval(() => {
      GetActivityStatus().then((s) => setLive(s || null)).catch(() => {});
    }, 8000);
    return () => { off?.(); clearInterval(id); };
  }, [open, range, refDay, appFilter]);

  if (!open) return null;

  const maxApp = Math.max(1, ...(summary?.byApp || []).map((x) => x.seconds));
  const maxDom = Math.max(1, ...(summary?.byDomain || []).map((x) => x.seconds));
  const maxDis = Math.max(1, ...(summary?.topDistractions || []).map((x) => x.seconds));

  return (
    <Sheet open={open} onClose={onClose} className="max-w-3xl">
      <SheetHeader title={t.insights} onClose={onClose} />
      <SheetBody>
        <div className="mb-3 flex flex-wrap items-center gap-1.5">
          {[
            { k: 'day', label: t.rangeDay },
            { k: 'week', label: t.rangeWeek },
            { k: 'month', label: t.rangeMonth },
            { k: 'overall', label: t.rangeOverall },
          ].map((o) => (
            <button
              key={o.k}
              onClick={() => setRange(o.k)}
              className={cn(
                'rounded-lg border px-3 py-1.5 text-xs font-bold transition-colors',
                range === o.k ? 'border-primary/50 bg-primary/15 text-primary' : 'border-border text-muted-foreground hover:text-foreground'
              )}
            >
              {o.label}
            </button>
          ))}
          {range === 'day' && (
            <div className="ms-auto flex items-center gap-1">
              <button title={t.prevDay} onClick={() => setRefDay((d) => shiftDay(d, -1))} className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground">
                <ChevronLeft size={15} />
              </button>
              <button onClick={() => setRefDay(dayStr(new Date()))} className="tabular rounded-md px-2 py-1 text-xs font-bold text-muted-foreground hover:bg-accent hover:text-foreground">
                {refDay} · {t.todayBtn}
              </button>
              <button title={t.nextDay} onClick={() => setRefDay((d) => shiftDay(d, 1))} className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground">
                <ChevronRight size={15} />
              </button>
            </div>
          )}
        </div>

        {live?.enabled && live?.app && range === 'day' && (
          <div className="mb-3 flex items-center gap-2 rounded-xl border border-emerald-500/30 bg-emerald-500/5 px-3 py-2 text-xs">
            <span className="relative flex size-2 shrink-0">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-60" />
              <span className="relative inline-flex size-2 rounded-full bg-emerald-400" />
            </span>
            <span className="text-muted-foreground">{t.liveNow}:</span>
            <span className="min-w-0 flex-1 truncate font-bold">{live.detail || live.title || live.app}</span>
            <span className={cn('size-1.5 shrink-0 rounded-full', catDot(live.category))} />
            <span className="tabular shrink-0 text-muted-foreground">{formatHMS(live.elapsedSeconds || 0)}</span>
          </div>
        )}

        {!enabled && !summary?.totalSeconds ? (
          <div className="rounded-xl border border-dashed p-6 text-center">
            <Radio size={22} className="mx-auto text-muted-foreground" />
            <p className="mt-2 text-sm font-bold">{t.noActivity}</p>
            <p className="mx-auto mt-1 max-w-sm text-xs leading-relaxed text-muted-foreground">{t.noActivitySub}</p>
          </div>
        ) : (
          <>
            <div className="flex flex-wrap gap-2">
              <StatCard icon={Clock} label={`${t.totalTracked2} · ${summary?.sessions || 0} ${t.sessionsLabel}`} secs={summary?.totalSeconds || 0} tone="text-primary" />
              <StatCard icon={Briefcase} label={t.workTime} secs={summary?.workSeconds || 0} tone="text-emerald-400" />
              <StatCard icon={Flame} label={`${t.wastedTime} · ${t.wastedOf(summary?.wastedPct || 0)}`} secs={summary?.distractionSeconds || 0} tone="text-rose-400" />
              <StatCard icon={Shapes} label={t.otherTime} secs={summary?.otherSeconds || 0} tone="text-sky-400" />
              <StatCard icon={Moon} label={t.idleTime} secs={summary?.idleSeconds || 0} tone="text-muted-foreground" />
            </div>
            {loading && <p className="mt-2 text-xs text-muted-foreground">…</p>}

            {(summary?.topDistractions?.length > 0) && (
              <div className="mt-3 rounded-xl border border-rose-500/20 bg-rose-500/[0.03] p-3">
                <p className="mb-1 flex items-center gap-1.5 text-[13px] font-bold text-rose-300">
                  <Flame size={14} /> {t.topDistractions}
                </p>
                {(summary.topDistractions || []).map((r, i) => (
                  <Row key={i} name={r.name} secs={r.seconds} max={maxDis} sessions={r.sessions} accent="bg-rose-400" />
                ))}
              </div>
            )}

            <div className="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
              <div className="rounded-xl border p-3">
                <p className="mb-1 flex items-center gap-1.5 text-[13px] font-bold">
                  <BarChart3 size={14} className="text-primary" /> {t.topApps}
                </p>
                {(summary?.byApp || []).length === 0 && <p className="py-2 text-center text-xs text-muted-foreground">{t.noActivity}</p>}
                {(summary?.byApp || []).length > 0 && <p className="mb-1 text-[11px] text-muted-foreground">{t.tapToFilter}</p>}
                {(summary?.byApp || []).map((r, i) => (
                  <Row
                    key={i} name={r.name} secs={r.seconds} max={maxApp} sessions={r.sessions}
                    accent={r.category === 'work' ? 'bg-emerald-400' : r.category === 'distraction' ? 'bg-rose-400' : r.category === 'idle' ? 'bg-muted-foreground/50' : 'bg-sky-400'}
                    selected={appFilter === r.name}
                    onClick={() => setAppFilter((cur) => (cur === r.name ? 'all' : r.name))}
                  />
                ))}
              </div>
              <div className="rounded-xl border p-3">
                <p className="mb-1 flex items-center gap-1.5 text-[13px] font-bold">
                  <Globe size={14} className="text-primary" /> {t.topSites}
                </p>
                {(summary?.byDomain || []).length === 0 && <p className="py-2 text-center text-xs text-muted-foreground">—</p>}
                {(summary?.byDomain || []).map((r, i) => (
                  <Row
                    key={i} name={r.name} secs={r.seconds} max={maxDom} sessions={r.sessions}
                    accent={r.category === 'distraction' ? 'bg-rose-400' : 'bg-emerald-400'}
                  />
                ))}
              </div>
            </div>

            <div className="mt-3 rounded-xl border p-3">
              <div className="mb-2 flex flex-wrap items-center gap-2">
                <p className="flex min-w-0 flex-1 items-center gap-1.5 text-[13px] font-bold">
                  <ListVideo size={14} className="shrink-0 text-primary" /> {t.recentActivity}
                </p>
                <Select
                  value={appFilter}
                  onChange={(e) => setAppFilter(e.target.value)}
                  className="h-8 w-40 py-1 text-xs font-semibold"
                >
                  <option value="all">{t.allApps}</option>
                  {(summary?.byApp || []).map((r) => (
                    <option key={r.name} value={r.name}>{r.name}</option>
                  ))}
                </Select>
                <div className="relative">
                  <Input
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder={t.searchActivity}
                    className="h-8 w-44 pe-7 text-xs"
                  />
                  {search && (
                    <button onClick={() => setSearch('')} className="absolute end-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
                      <X size={13} />
                    </button>
                  )}
                </div>
              </div>
              {(() => {
                const q = search.trim().toLowerCase();
                const visible = (segments || []).filter((s) => !q
                  || `${s.detail || ''} ${s.title || ''} ${s.domain || ''} ${s.app || ''}`.toLowerCase().includes(q));
                const total = visible.reduce((a, s) => a + (s.seconds || 0), 0);
                return (
                  <>
                    {(appFilter !== 'all' || q) && (
                      <div className="mb-1.5 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
                        <span className="tabular font-bold text-foreground">
                          {appFilter !== 'all' ? t.inAppTotal(appFilter, formatHMS(total)) : formatHMS(total)}
                        </span>
                        <span>·</span>
                        <span>{t.showingOf(visible.length, (segments || []).length)}</span>
                        <button
                          onClick={() => { setAppFilter('all'); setSearch(''); }}
                          className="font-bold text-primary hover:underline"
                        >
                          {t.clear}
                        </button>
                      </div>
                    )}
                    <div className="max-h-64 space-y-1 overflow-y-auto">
                      {visible.map((s) => (
                        <div key={s.id} className="flex items-center gap-2 rounded-lg px-2 py-1.5 hover:bg-accent/50">
                          <span className={cn('size-2 shrink-0 rounded-full', catDot(s.category))} />
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-xs font-semibold" title={s.detail || s.title}>
                              {s.detail || s.title || s.app}
                              {s.domain === 'youtube.com' && s.detail ? <span className="font-normal text-muted-foreground"> · {t.youtubeWatched}</span> : null}
                            </p>
                            <p className="truncate text-[11px] text-muted-foreground">
                              {s.app}{s.domain ? ` · ${s.domain}` : ''} · {fmtTime(s.startedAt)}
                            </p>
                          </div>
                          <span className="tabular shrink-0 text-xs text-muted-foreground">{formatHMS(s.seconds)}</span>
                        </div>
                      ))}
                      {visible.length === 0 && <p className="py-2 text-center text-xs text-muted-foreground">{t.noActivity}</p>}
                    </div>
                  </>
                );
              })()}
            </div>
          </>
        )}
      </SheetBody>
      <SheetFooter>
        <Button variant="ghost" onClick={onClose}><X /> {t.close}</Button>
      </SheetFooter>
    </Sheet>
  );
}
