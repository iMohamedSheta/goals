import * as React from 'react';
import { Search, Plus, Star, X, SlidersHorizontal, RotateCcw, Bot } from 'lucide-react';
import { cn } from './lib/utils';
import { Button } from './components/ui/button';
import { Badge } from './components/ui/badge';
import { Input } from './components/ui/form';
import { Sidebar } from './components/Sidebar';
import { Menubar, WindowControls } from './components/Menubar';
import { KanbanBoard, TaskTable, ViewToggle, STATUSES, ActiveTimerPill, MiniTimer, TimerWidget } from './components/Board';
import { TaskSheet, ConfirmModal } from './components/Dialogs';
import { ChatDrawer } from './components/ChatDrawer';
import { SettingsSheet } from './components/SettingsSheet';
import { STR, formatHMS, liveElapsed, horizonName, horizonDesc } from './lib/i18n';
import {
  DEFAULT_APPEARANCE, loadLocalAppearance, saveLocalAppearance, applyAppearance,
  fromSettingsMap, settingsDiff, filterChipClass, density, animClass, RADIUS,
} from './lib/appearance';
import {
  ListTasks, GetStats, ListContexts, CreateContext, UpdateContext, DeleteContext,
  ListHorizons, UpdateHorizon, CreateHorizon, DeleteHorizon,
  CreateTask, UpdateTask, MoveTask, ToggleFocus, DeleteTask, GetDBPath,
  StartTimer, StopTimer, FinishTask, GetActiveTimer, ListTimeEntries, QuitApp,
  GetSettings, SetSetting, OpenDataFolder, PickDatabaseFile, ExePath,
  GetDriveStatus, SaveDriveCredentials, StartDriveAuth, PollDriveAuth, CancelDriveAuth, DisconnectDrive,
  BackupNow, ListDriveBackups, DeleteDriveBackup, RestoreDriveBackup, ImportDatabaseFile,
  GetVersion, CheckForUpdates, DownloadUpdate, InstallUpdateAndRestart,
} from '../wailsjs/go/main/App';
import { EventsOn, WindowSetSize, WindowSetMinSize, WindowSetAlwaysOnTop, WindowSetPosition, ScreenGetAll, WindowReload, WindowUnfullscreen, WindowUnmaximise, WindowMaximise, WindowIsFullscreen, WindowIsMaximised } from '../wailsjs/runtime/runtime';

export default function App() {
  const [lang, setLang] = React.useState(() => localStorage.getItem('goals-lang') || 'ar');
  const t = STR[lang] || STR.ar;

  const [horizons, setHorizons] = React.useState([]);
  const [contexts, setContexts] = React.useState([]);
  const [tasks, setTasks] = React.useState([]);
  const [stats, setStats] = React.useState(null);
  const [counts, setCounts] = React.useState({});
  const [dbPath, setDbPath] = React.useState('');
  const [exePath, setExePath] = React.useState('');
  const [version, setVersion] = React.useState('');
  const [latestTag, setLatestTag] = React.useState('');
  const [ready, setReady] = React.useState(false);
  const [error, setError] = React.useState(null);

  const [activeHorizon, setActiveHorizon] = React.useState('short');
  const [view, setView] = React.useState('kanban');
  const [focusOnly, setFocusOnly] = React.useState(false);
  const [contextFilter, setContextFilter] = React.useState('all');
  const [statusFilter, setStatusFilter] = React.useState('all');
  const [search, setSearch] = React.useState('');

  const [sheet, setSheet] = React.useState({ open: false, task: null, presetStatus: 'todo' });
  const [entries, setEntries] = React.useState([]);
  const [settings, setSettings] = React.useState({ open: false, tab: 'appearance' });
  const openSettings = (tab) => setSettings({ open: true, tab: tab || 'appearance' });
  const [confirm, setConfirm] = React.useState({ open: false, task: null });
  const [finishConfirm, setFinishConfirm] = React.useState({ open: false, task: null, secs: 0 });
  const [restoreConfirm, setRestoreConfirm] = React.useState({ open: false, id: null, name: '' });

  const [active, setActive] = React.useState(null); // {id,title,elapsed,timerStartedAt,maxSeconds}
  const [miniMode, setMiniMode] = React.useState(null); // null | 'widget' | 'side'
  const [chatOpen, setChatOpen] = React.useState(false);

  const [appearance, setAppearance] = React.useState(() => loadLocalAppearance());
  const appearanceRef = React.useRef(appearance);
  const appearanceDirty = JSON.stringify(appearance) !== JSON.stringify(DEFAULT_APPEARANCE);

  const updateAppearance = React.useCallback(async (patch) => {
    const next = { ...appearanceRef.current, ...patch };
    appearanceRef.current = next;
    setAppearance(next);
    applyAppearance(next);
    saveLocalAppearance(next);
    try {
      for (const [k, v] of settingsDiff(appearance, next)) await SetSetting(k, v);
    } catch { /* backend offline — local still applied */ }
  }, [appearance]);

  const resetAppearance = React.useCallback(async () => {
    appearanceRef.current = { ...DEFAULT_APPEARANCE };
    setAppearance({ ...DEFAULT_APPEARANCE });
    applyAppearance(DEFAULT_APPEARANCE);
    saveLocalAppearance(DEFAULT_APPEARANCE);
    try {
      for (const [k, v] of settingsDiff(appearance, DEFAULT_APPEARANCE)) await SetSetting(k, v);
    } catch { /* noop */ }
  }, [appearance]);

  const searchRef = React.useRef(null);
  const contextOf = React.useCallback((id) => contexts.find((c) => c.id === id), [contexts]);
  const horizon = horizons.find((h) => h.key === activeHorizon);
  const focusedCount = stats?.focused || 0;

  React.useEffect(() => {
    document.documentElement.lang = lang;
    document.documentElement.dir = t.dir;
    localStorage.setItem('goals-lang', lang);
  }, [lang]); // eslint-disable-line

  const syncActive = React.useCallback(async () => {
    try {
      const a = await GetActiveTimer();
      setActive({ id: a.id, title: a.title, elapsed: liveElapsed(a, Date.now()), timerStartedAt: a.timerStartedAt, maxSeconds: a.maxSeconds || 0 });
    } catch {
      setActive(null);
    }
  }, []);

  const refresh = React.useCallback(async () => {
    const f = { horizon: 'all', status: 'all', contextId: 'all', focusOnly: false, search: '' };
    if (focusOnly) f.focusOnly = true;
    if (contextFilter !== 'all') f.contextId = contextFilter;
    if (statusFilter !== 'all') f.status = statusFilter;
    if (search.trim()) f.search = search.trim();
    const [all, board, st] = await Promise.all([
      ListTasks({ horizon: 'all', status: 'all', contextId: 'all', focusOnly: false, search: '' }),
      ListTasks(f),
      GetStats(),
    ]);
    const c = {};
    for (const h of horizons.length ? horizons.map((x) => x.key) : ['short', 'medium', 'long']) {
      const items = (all || []).filter((x) => x.horizon === h);
      c[h] = { total: items.length, done: items.filter((x) => x.status === 'done').length };
    }
    setCounts(c);
    setTasks(board || []);
    setStats(st);
    await syncActive();
  }, [focusOnly, contextFilter, statusFilter, search, syncActive, horizons]);

  React.useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(() => {
      if (!cancelled) {
        setReady((r) => {
          if (!r) setError((e) => e || t.bootTimeout);
          return r;
        });
      }
    }, 15000);
    (async () => {
      try {
        const [hs, cs, db, backendSettings] = await Promise.all([ListHorizons(), ListContexts(), GetDBPath(), GetSettings().catch(() => ({}))]);
        if (cancelled) return;
        setHorizons(hs || []);
        setContexts(cs || []);
        setDbPath(db || '');
        try {
          setExePath(await ExePath());
        } catch { /* noop */ }
        try {
          setVersion(await GetVersion());
        } catch { /* noop */ }
        // silent update check, at most once a day — never blocks boot
        try {
          const last = +(localStorage.getItem('goals-last-update-check') || 0);
          if (Date.now() - last > 24 * 3600 * 1000) {
            CheckForUpdates(false).then((st) => {
              try { localStorage.setItem('goals-last-update-check', String(Date.now())); } catch { /* noop */ }
              if (!cancelled && st?.available) setLatestTag(st.latest || '');
            }).catch(() => { /* offline — stay silent */ });
          }
        } catch { /* noop */ }
        if ((hs || []).length) {
          setActiveHorizon((cur) => ((hs || []).some((h) => h.key === cur) ? cur : hs[0].key));
        }
        const merged = fromSettingsMap(backendSettings);
        // backend wins: it holds every key the user ever saved
        const hasBackend = backendSettings && Object.keys(backendSettings).some((k) => k.startsWith('appearance.'));
        if (hasBackend) {
          appearanceRef.current = merged;
          setAppearance(merged);
          applyAppearance(merged);
          saveLocalAppearance(merged);
        } else {
          applyAppearance(appearanceRef.current);
        }
        await syncActive();
        if (cancelled) return;
        clearTimeout(timer);
        setReady(true);
      } catch (err) {
        if (!cancelled) {
          clearTimeout(timer);
          setError(String(err));
        }
      }
    })();
    return () => { cancelled = true; clearTimeout(timer); };
  }, [syncActive]); // eslint-disable-line

  React.useEffect(() => {
    if (ready) refresh().catch((e) => setError(String(e)));
  }, [ready, refresh]);

  // live backend ticks (1s while a timer runs)
  React.useEffect(() => {
    const off = EventsOn('timer:tick', (payload) => {
      if (!payload) {
        setActive(null);
        return;
      }
      setActive((prev) => ({
        id: payload.taskId,
        title: payload.title,
        elapsed: payload.elapsed,
        timerStartedAt: prev && prev.id === payload.taskId ? prev.timerStartedAt : prev?.timerStartedAt,
        maxSeconds: prev && prev.id === payload.taskId ? prev.maxSeconds : prev?.maxSeconds,
      }));
    });
    return () => off?.();
  }, []);

  // debounce search
  const [debounced, setDebounced] = React.useState('');
  React.useEffect(() => {
    const id = setTimeout(() => setDebounced(search), 250);
    return () => clearTimeout(id);
  }, [search]);
  React.useEffect(() => {
    if (!ready) return;
    const f = { horizon: 'all', status: 'all', contextId: 'all', focusOnly: false, search: '' };
    if (focusOnly) f.focusOnly = true;
    if (contextFilter !== 'all') f.contextId = contextFilter;
    if (statusFilter !== 'all') f.status = statusFilter;
    if (debounced.trim()) f.search = debounced.trim();
    ListTasks(f).then((b) => setTasks(b || [])).catch(() => {});
  }, [debounced]); // eslint-disable-line

  React.useEffect(() => {
    const onKey = (e) => {
      if (miniMode) return;
      const typing = ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName);
      if (e.key === '/' && !typing) { e.preventDefault(); searchRef.current?.focus(); }
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setSheet({ open: true, task: null, presetStatus: 'todo' });
      }
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'j') {
        e.preventDefault();
        setChatOpen((v) => !v);
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [miniMode]);

  // ---------- timer actions ----------

  const startTimer = async (task) => {
    await StartTimer(task.id);
    await refresh();
  };
  const pauseTimer = async (taskOrId) => {
    const id = typeof taskOrId === 'string' ? taskOrId : taskOrId.id;
    await StopTimer(id);
    await refresh();
  };
  const askFinish = (task) => {
    const secs = liveElapsed(task, Date.now());
    setFinishConfirm({ open: true, task, secs });
  };
  const doFinish = async () => {
    await FinishTask(finishConfirm.task.id);
    setFinishConfirm({ open: false, task: null, secs: 0 });
    if (miniMode) exitMini();
    await refresh();
  };

  const MINI_SIZES = {
    widget: { w: 216, h: 54, minW: 200, minH: 48 },
    side: { w: 320, h: 470, minW: 300, minH: 420 },
  };

  // dock the mini window to the screen's end edge, vertically centered
  // A maximized/fullscreen window ignores WindowSetSize, so always step out
  // of fullscreen/maximized first (with a beat for the OS to apply it).
  const safeFlag = (fn) => {
    try {
      return Promise.resolve(fn()).catch(() => false);
    } catch {
      return Promise.resolve(false);
    }
  };
  const leaveFullWindow = async () => {
    const [fs, mx] = await Promise.all([safeFlag(WindowIsFullscreen), safeFlag(WindowIsMaximised)]);
    if (fs) { try { WindowUnfullscreen(); } catch { /* noop */ } }
    if (mx) { try { WindowUnmaximise(); } catch { /* noop */ } }
    if (fs || mx) await new Promise((r) => setTimeout(r, 180));
    return { wasFullscreen: fs, wasMaximised: mx };
  };
  // remembers how the main window looked before entering mini, so exitMini
  // can put it back (e.g. re-maximize if it was maximized)
  const miniReturnRef = React.useRef({ wasMaximised: false });
  const applyMiniWindow = async (mode) => {
    const s = MINI_SIZES[mode];
    if (!s) return;
    try {
      await leaveFullWindow();
      WindowSetAlwaysOnTop(true);
      WindowSetMinSize(s.minW, s.minH);
      WindowSetSize(s.w, s.h);
      const screens = await ScreenGetAll().catch(() => []);
      const scr = (screens || []).find((x) => x.isPrimary) || (screens || [])[0];
      const W = scr?.size?.width || scr?.width;
      const H = scr?.size?.height || scr?.height;
      if (W && H) {
        const rtl = document.documentElement.dir === 'rtl';
        const x = rtl ? 12 : Math.max(0, W - s.w - 12);
        const y = Math.max(0, Math.round((H - s.h) / 2));
        WindowSetPosition(x, y);
      }
    } catch { /* noop */ }
  };

  const enterMini = async () => {
    if (!active) return;
    // capture pre-mini state before applyMiniWindow clears it
    try {
      const [fs, mx] = await Promise.all([safeFlag(WindowIsFullscreen), safeFlag(WindowIsMaximised)]);
      miniReturnRef.current = { wasMaximised: !!(fs || mx) };
    } catch { /* noop */ }
    setMiniMode('widget');
    applyMiniWindow('widget');
  };
  const expandWidget = () => {
    setMiniMode('side');
    applyMiniWindow('side');
  };
  const collapseWidget = () => {
    setMiniMode('widget');
    applyMiniWindow('widget');
  };
  const exitMini = () => {
    setMiniMode(null);
    try {
      WindowSetAlwaysOnTop(false);
      WindowSetSize(1280, 800);
      WindowSetMinSize(940, 600);
      if (miniReturnRef.current.wasMaximised) {
        // let the resize land first, then restore maximized state
        setTimeout(() => { try { WindowMaximise(); } catch { /* noop */ } }, 120);
        miniReturnRef.current = { wasMaximised: false };
      }
    } catch { /* noop */ }
  };

  const quitApp = async () => {
    try {
      await QuitApp();
    } catch {
      try {
        if (active) await StopTimer(active.id);
      } catch { /* already stopped */ }
    }
  };

  const openTaskSheet = async (task, presetStatus) => {
    if (task) {
      try {
        setEntries(await ListTimeEntries(task.id));
      } catch {
        setEntries([]);
      }
    } else {
      setEntries([]);
    }
    setSheet({ open: true, task: task || null, presetStatus: presetStatus || 'todo' });
  };

  const saveTask = async (input) => {
    if (sheet.task) await UpdateTask(sheet.task.id, input);
    else {
      const created = await CreateTask(input);
      if (created?.horizon) setActiveHorizon(created.horizon);
    }
    setSheet({ open: false, task: null, presetStatus: 'todo' });
    await refresh();
  };

  const reloadHorizons = async () => {
    const hs = await ListHorizons();
    setHorizons(hs || []);
    setActiveHorizon((cur) => ((hs || []).some((h) => h.key === cur) ? cur : (hs[0]?.key || cur)));
    await refresh();
  };

  const saveHorizons = async (rows) => {
    for (const r of rows) {
      await UpdateHorizon(r.key, r.label, r.labelAr || '', r.defaultDays, r.description, r.descriptionAr || '');
    }
    await reloadHorizons();
  };

  const createHorizon = async (draft) => {
    try {
      const h = await CreateHorizon(draft.label.trim(), (draft.labelAr || '').trim(), draft.days || 30, '', '');
      await reloadHorizons();
      setActiveHorizon(h.key);
      return { ok: true };
    } catch (err) {
      return { ok: false, error: String(err) };
    }
  };

  const deleteHorizon = async (h) => {
    try {
      await DeleteHorizon(h.key);
      await reloadHorizons();
      return { ok: true };
    } catch (err) {
      return { ok: false, error: String(err) };
    }
  };

  const importLocalDb = async () => {
    try {
      const path = await PickDatabaseFile();
      if (!path) return { ok: true };
      await ImportDatabaseFile(path);
      WindowReload();
      return { ok: true };
    } catch (err) {
      return { ok: false, error: String(err) };
    }
  };

  const doRestore = async () => {
    try {
      await RestoreDriveBackup(restoreConfirm.id);
      WindowReload();
    } catch (err) {
      setRestoreConfirm({ open: false, id: null, name: '' });
      alert(String(err));
    }
  };

  // ---------- mini mode ----------

  if (miniMode) {
    return (
      <div className="flex h-full flex-col overflow-hidden bg-background">
        <div className="min-h-0 flex-1">
          {miniMode === 'widget' ? (
            <TimerWidget
              t={t}
              active={active}
              onExpand={expandWidget}
            />
          ) : (
            <MiniTimer
              t={t}
              active={active}
              onPause={() => active && pauseTimer(active.id)}
              onResume={() => active && startTimer(active)}
              onFinish={(a) => askFinish(a.id ? { id: a.id, title: a.title, elapsedSeconds: a.elapsed, timerStartedAt: a.timerStartedAt } : a)}
              onExpand={exitMini}
              onCollapse={collapseWidget}
            />
          )}
        </div>
        <ConfirmModal
          t={t}
          open={finishConfirm.open}
          onClose={() => setFinishConfirm({ open: false, task: null, secs: 0 })}
          title={t.finishTitle}
          message={t.finishMsg(finishConfirm.task?.title || '', formatHMS(finishConfirm.secs))}
          confirmLabel={t.finishBtn}
          danger={false}
          onConfirm={doFinish}
        />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-full flex-col items-center justify-center overflow-hidden bg-background">
        <div className="absolute end-0 top-0" style={{ ['--wails-draggable']: 'drag' }}>
          <WindowControls t={t} small />
        </div>
        <div className="max-w-md rounded-xl border border-red-500/30 bg-red-500/5 p-6 text-center">
          <h2 className="font-semibold text-red-400">{t.failTitle}</h2>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{error}</p>
          <Button className="mt-4" onClick={() => WindowReload()}>
            <RotateCcw /> {t.retry}
          </Button>
        </div>
      </div>
    );
  }
  if (!ready) {
    return (
      <div className="flex h-full flex-col items-center justify-center overflow-hidden bg-background">
        <div className="absolute end-0 top-0" style={{ ['--wails-draggable']: 'drag' }}>
          <WindowControls t={t} small />
        </div>
        <div className="flex items-center gap-3 text-sm text-muted-foreground">
          <span className="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          {t.loading}
        </div>
      </div>
    );
  }

  const boardTasks = tasks.filter((x) => x.horizon === activeHorizon);
  const hasFilters = focusOnly || contextFilter !== 'all' || statusFilter !== 'all' || search.trim();
  const clearFilters = () => { setFocusOnly(false); setContextFilter('all'); setStatusFilter('all'); setSearch(''); };

  return (
    <div className="flex h-full flex-col overflow-hidden bg-background">
      <Menubar
        t={t}
        lang={lang}
        horizons={horizons}
        activeHorizon={activeHorizon}
        view={view}
        focusOnly={focusOnly}
        hasFilters={hasFilters}
        timerRunning={!!active}
        onNewTask={() => openTaskSheet(null, 'todo')}
        onView={setView}
        onHorizon={(h) => { setActiveHorizon(h); setFocusOnly(false); }}
        onFocusToggle={() => setFocusOnly((v) => !v)}
        onClearFilters={clearFilters}
        onMini={enterMini}
        onSettings={(tab) => openSettings(tab)}
        onLangToggle={() => setLang((l) => (l === 'ar' ? 'en' : 'ar'))}
        onQuit={quitApp}
      />
      <div className="flex min-h-0 flex-1">
      <Sidebar
        t={t}
        lang={lang}
        ap={appearance}
        onLangToggle={() => setLang((l) => (l === 'ar' ? 'en' : 'ar'))}
        horizons={horizons}
        contexts={contexts}
        counts={counts}
        focusedCount={focusedCount}
        stats={stats}
        activeHorizon={activeHorizon}
        onHorizon={(h) => { setActiveHorizon(h); setFocusOnly(false); }}
        focusOnly={focusOnly}
        onFocusToggle={() => setFocusOnly((v) => !v)}
        contextFilter={contextFilter}
        onContextFilter={setContextFilter}
        dbPath={dbPath}
        timerRunning={!!active}
        onQuit={quitApp}
        onSettings={() => openSettings('appearance')}
        onOpenFolder={async () => { try { await OpenDataFolder(); } catch { /* noop */ } }}
        version={version}
        updateAvailable={!!latestTag}
        onOpenUpdates={() => openSettings('data')}
      />

      <main className="flex min-w-0 flex-1 flex-col">
        <header className="border-b bg-card/30 px-6 pb-4 pt-5">
          <div className="flex flex-wrap items-center gap-3">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2.5">
                <h2 className="truncate text-xl font-bold tracking-tight">{horizonName(horizon, lang) || 'Goals'}</h2>
                {horizon && (
                  <Badge variant="secondary" className={cn('tabular shrink-0', density(appearance).badge)}>~{horizon.defaultDays}</Badge>
                )}
                {focusOnly && (
                  <Badge variant="warning" className={density(appearance).badge}><Star size={12} fill="currentColor" /> {t.focusedOnly}</Badge>
                )}
              </div>
              <p className="mt-0.5 truncate text-[13px] text-muted-foreground">{horizonDesc(horizon, lang)}</p>
            </div>
            {active && (
              <ActiveTimerPill
                t={t}
                active={active}
                ap={appearance}
                onPause={() => pauseTimer(active.id)}
                onResume={() => startTimer(active)}
                onMini={enterMini}
                onFinish={() => askFinish({ id: active.id, title: active.title, elapsedSeconds: active.elapsed, timerStartedAt: active.timerStartedAt })}
              />
            )}
            <div className="relative">
              <Search size={15} className="pointer-events-none absolute start-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
              <Input
                ref={searchRef}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t.searchPh}
                className="w-52 ps-9"
              />
              {search && (
                <button onClick={() => setSearch('')} className="absolute end-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground">
                  <X size={14} />
                </button>
              )}
            </div>
            <ViewToggle t={t} view={view} onChange={setView} ap={appearance} />
            <Button onClick={() => openTaskSheet(null, 'todo')}>
              <Plus /> {t.newTask}
            </Button>
          </div>

          <div className="mt-3.5 flex flex-wrap items-center gap-1.5">
            <SlidersHorizontal size={13} className="me-1 text-muted-foreground" />
            {[{ key: 'all' }, ...STATUSES].map((s) => (
              <button
                key={s.key}
                onClick={() => setStatusFilter(s.key)}
                className={cn(filterChipClass(appearance, statusFilter === s.key))}
              >
                {t[s.key]}
              </button>
            ))}
            <span className="tabular ms-auto text-xs text-muted-foreground">
              {t.shownDone(boardTasks.length, boardTasks.filter((x) => x.status === 'done').length)}
            </span>
            {hasFilters && (
              <button onClick={clearFilters} className={cn('flex items-center gap-1 font-medium text-muted-foreground hover:text-foreground', RADIUS, density(appearance).chip, animClass(appearance))}>
                <X size={12} /> {t.clear}
              </button>
            )}
          </div>
        </header>

        <div className="flex min-h-0 flex-1 flex-col p-5">
          {view === 'kanban' ? (
            <KanbanBoard
              t={t}
              tasks={boardTasks}
              ap={appearance}
              contextOf={contextOf}
              onEdit={(x) => openTaskSheet(x, x.status)}
              onDelete={(x) => setConfirm({ open: true, task: x })}
              onToggleFocus={async (x) => { await ToggleFocus(x.id); await refresh(); }}
              onStart={startTimer}
              onPause={pauseTimer}
              onFinish={askFinish}
              onDrop={async (id, status) => { await MoveTask(id, status); await refresh(); }}
              onCreateFirst={() => openTaskSheet(null, 'todo')}
            />
          ) : (
            <div className="overflow-y-auto">
              <TaskTable
                t={t}
                tasks={boardTasks}
                ap={appearance}
                contextOf={contextOf}
                onEdit={(x) => openTaskSheet(x, x.status)}
                onDelete={(x) => setConfirm({ open: true, task: x })}
                onToggleFocus={async (x) => { await ToggleFocus(x.id); await refresh(); }}
                onStart={startTimer}
                onPause={pauseTimer}
                onFinish={askFinish}
              />
            </div>
          )}
        </div>
      </main>
      </div>

      <TaskSheet
        t={t}
        open={sheet.open}
        onClose={() => setSheet({ open: false, task: null, presetStatus: 'todo' })}
        task={sheet.task}
        horizons={horizons}
        contexts={contexts}
        activeHorizon={activeHorizon}
        presetStatus={sheet.presetStatus}
        onSave={saveTask}
        entries={entries}
      />
      <SettingsSheet
        t={t}
        lang={lang}
        open={settings.open}
        onClose={() => setSettings((s) => ({ ...s, open: false }))}
        tab={settings.tab}
        onTab={(tab) => setSettings((s) => ({ ...s, tab }))}
        value={appearance}
        dirty={appearanceDirty}
        onChange={updateAppearance}
        onReset={resetAppearance}
        horizons={horizons}
        onSaveHorizons={saveHorizons}
        onCreateHorizon={createHorizon}
        onDeleteHorizon={deleteHorizon}
        contexts={contexts}
        onCreateContext={async (n, c) => { await CreateContext(n, c); setContexts(await ListContexts()); }}
        onUpdateContexts={async (drafts) => {
          for (const d of drafts) {
            const orig = contexts.find((x) => x.id === d.id);
            if (orig && (orig.name !== d.name || orig.color !== d.color)) await UpdateContext(d.id, d.name, d.color);
          }
          setContexts(await ListContexts());
          await refresh();
        }}
        onDeleteContext={async (d) => {
          await DeleteContext(d.id);
          setContexts(await ListContexts());
          if (contextFilter === d.id) setContextFilter('all');
          await refresh();
        }}
        dbPath={dbPath}
        onOpenFolder={async () => { try { await OpenDataFolder(); } catch { /* noop */ } }}
        exePath={exePath}
        version={version}
        latestTag={latestTag}
        onUpdateFound={(tag) => setLatestTag(tag || '')}
        updater={{
          check: (force) => CheckForUpdates(force),
          download: () => DownloadUpdate(),
          install: () => InstallUpdateAndRestart(),
        }}
        drive={{
          getStatus: GetDriveStatus,
          saveCreds: SaveDriveCredentials,
          startAuth: StartDriveAuth,
          pollAuth: PollDriveAuth,
          cancelAuth: CancelDriveAuth,
          disconnect: DisconnectDrive,
          backup: BackupNow,
          list: ListDriveBackups,
          del: DeleteDriveBackup,
        }}
        onImportLocal={importLocalDb}
        onAskRestore={(id, name) => setRestoreConfirm({ open: true, id, name })}
      />
      <ConfirmModal
        t={t}
        open={restoreConfirm.open}
        onClose={() => setRestoreConfirm({ open: false, id: null, name: '' })}
        title={t.restoreTitle}
        message={t.restoreMsg(restoreConfirm.name || '')}
        confirmLabel={t.restore}
        onConfirm={doRestore}
      />
      <ConfirmModal
        t={t}
        open={confirm.open}
        onClose={() => setConfirm({ open: false, task: null })}
        title={t.delTitle}
        message={t.delMsg(confirm.task?.title || '')}
        confirmLabel={t.delBtn}
        onConfirm={async () => { await DeleteTask(confirm.task.id); setConfirm({ open: false, task: null }); await refresh(); }}
      />
      <ConfirmModal
        t={t}
        open={finishConfirm.open}
        onClose={() => setFinishConfirm({ open: false, task: null, secs: 0 })}
        title={t.finishTitle}
        message={t.finishMsg(finishConfirm.task?.title || '', formatHMS(finishConfirm.secs))}
        confirmLabel={t.finishBtn}
        danger={false}
        onConfirm={doFinish}
      />
      <ChatDrawer
        t={t}
        open={chatOpen}
        onClose={() => setChatOpen(false)}
        onTasksChanged={refresh}
      />
      {!chatOpen && (
        <button
          onClick={() => setChatOpen(true)}
          title={`${t.aiChat} (Ctrl+J)`}
          className="fixed bottom-5 end-5 z-40 grid size-12 place-items-center rounded-full bg-primary text-primary-foreground shadow-xl transition-transform hover:scale-105 active:scale-95"
        >
          <Bot size={22} />
        </button>
      )}
    </div>
  );
}
