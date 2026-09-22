import * as React from 'react';
import { Search, Plus, Star, X, SlidersHorizontal, RotateCcw, Bot } from 'lucide-react';
import { cn } from './lib/utils';
import { Button } from './components/ui/button';
import { Badge } from './components/ui/badge';
import { Input } from './components/ui/form';
import { Sidebar } from './components/Sidebar';
import { Menubar, WindowControls } from './components/Menubar';
import { KanbanBoard, TaskTable, ViewToggle, STATUSES, ActiveTimerPill, ActiveContextPill, MiniTimer, TimerWidget } from './components/Board';
import { TaskSheet, ConfirmModal } from './components/Dialogs';
import { ChatDrawer } from './components/ChatDrawer';
import { SettingsSheet } from './components/SettingsSheet';
import { ActivityInsights } from './components/ActivityInsights';
import { PrayerAlert } from './components/PrayerAlert';
import { STR, formatHMS, liveElapsed, horizonName, horizonDesc } from './lib/i18n';
import {
  DEFAULT_APPEARANCE, loadLocalAppearance, saveLocalAppearance, applyAppearance,
  fromSettingsMap, settingsDiff, filterChipClass, density, animClass, RADIUS,
} from './lib/appearance';
import {
  ListTasks, GetStats, ListContexts, CreateContext, UpdateContext, DeleteContext,
  ListHorizons, UpdateHorizon, CreateHorizon, DeleteHorizon,
  CreateTask, UpdateTask, MoveTask, SetTaskParent, ReorderTasks, ToggleFocus, DeleteTask, GetDBPath,
  StartTimer, StopTimer, FinishTask, GetActiveTimer, GetActiveTimers, ListTimeEntries, QuitApp,
  StartContextTimer, StopContextTimer, GetActiveContextTimer,
  GetSettings, SetSetting, OpenDataFolder, PickDatabaseFile, ExePath,
  GetDriveStatus, SaveDriveCredentials, StartDriveAuth, PollDriveAuth, CancelDriveAuth, DisconnectDrive,
  BackupNow, ListDriveBackups, DeleteDriveBackup, RestoreDriveBackup, ImportDatabaseFile,
  GetVersion, CheckForUpdates, DownloadUpdate, InstallUpdateAndRestart,
  GetAppLanguage, SetAppLanguage, GetAutostartEnabled, SetAutostartEnabled,
  GetActivityEnabled, SetActivityEnabled, GetActivityStatus, ClearActivity,
  GetActivityRetention, SetActivityRetention, PruneActivityNow, GetActivityStats,
  GetPrayerSettings, SetPrayerSettings, GetPrayerStatus, PrayerSnooze, PrayerGoing,
} from '../wailsjs/go/main/App';
import { EventsOn, WindowSetSize, WindowSetMinSize, WindowSetAlwaysOnTop, WindowSetPosition, ScreenGetAll, WindowReload, WindowUnfullscreen, WindowUnmaximise, WindowMaximise, WindowIsFullscreen, WindowIsMaximised } from '../wailsjs/runtime/runtime';
import brandLogo from './assets/logo.png';

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

  const [sheet, setSheet] = React.useState({ open: false, task: null, presetStatus: 'todo', presetParentId: '' });
  const [entries, setEntries] = React.useState([]);
  const [settings, setSettings] = React.useState({ open: false, tab: 'general' });
  const openSettings = (tab) => setSettings({ open: true, tab: tab || 'general' });
  const [confirm, setConfirm] = React.useState({ open: false, task: null });
  const [finishConfirm, setFinishConfirm] = React.useState({ open: false, task: null, secs: 0 });
  const [restoreConfirm, setRestoreConfirm] = React.useState({ open: false, id: null, name: '' });

  const [active, setActive] = React.useState(null); // {id,title,elapsed,timerStartedAt,maxSeconds}
  const [activeCount, setActiveCount] = React.useState(1);
  const [miniMode, setMiniMode] = React.useState(null); // null | 'widget' | 'side'
  const [chatOpen, setChatOpen] = React.useState(false);

  // general settings: language (persisted) + OS autostart (default off)
  const [autostartOn, setAutostartOn] = React.useState(false);
  const [autostartBusy, setAutostartBusy] = React.useState(false);
  const [autostartErr, setAutostartErr] = React.useState('');

  // activity tracking (opt-in, default off) + insights
  const [insightsOpen, setInsightsOpen] = React.useState(false);
  const [activityEnabled, setActivityEnabled] = React.useState(false);
  const [activityBusy, setActivityBusy] = React.useState(false);
  const [activityAlertsOn, setActivityAlertsOn] = React.useState(true);
  const [activityDailyMin, setActivityDailyMin] = React.useState(30);
  const [activitySessionMin, setActivitySessionMin] = React.useState(10);
  const [activityLive, setActivityLive] = React.useState(null);
  const [activityClearMsg, setActivityClearMsg] = React.useState('');
  const [activityRetentionDays, setActivityRetentionDays] = React.useState(90);
  const [activityStats, setActivityStats] = React.useState(null);
  const [activityPruneMsg, setActivityPruneMsg] = React.useState('');
  const [distraction, setDistraction] = React.useState(null); // toast from backend alert

  // prayer times & alerts (opt-in, default off)
  const [prayerSettings, setPrayerSettings] = React.useState(null);
  const [prayerStatus, setPrayerStatus] = React.useState(null);
  const [prayerDue, setPrayerDue] = React.useState(null);
  const [prayerSavedFlash, setPrayerSavedFlash] = React.useState(false);

  // context goals: every context doubles as a focus goal with its own
  // timer running in parallel with the task timer
  const todayStr = () => new Date().toISOString().slice(0, 10);
  const [activeContext, setActiveContext] = React.useState(null); // {id,name,color,elapsed,timerStartedAt,maxSeconds,dailyTargetSeconds,recurrence,todaySeconds,weekSeconds}
  // last context seen running — lets the mini window offer resume after a stop
  const lastCtxRef = React.useRef(null);

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

  const changeLang = React.useCallback(async (l) => {
    const v = l === 'en' ? 'en' : 'ar';
    setLang(v);
    try { await SetAppLanguage(v); } catch { /* backend offline — localStorage still wins */ }
  }, []);

  const toggleAutostart = React.useCallback(async () => {
    setAutostartBusy(true);
    setAutostartErr('');
    try {
      const next = !autostartOn;
      await SetAutostartEnabled(next);
      setAutostartOn(next);
    } catch (e) {
      setAutostartErr(String(e?.message || e));
    } finally {
      setAutostartBusy(false);
    }
  }, [autostartOn]);

  const toggleActivity = React.useCallback(async () => {
    setActivityBusy(true);
    try {
      const next = !activityEnabled;
      await SetActivityEnabled(next);
      setActivityEnabled(next);
      if (next) {
        try { setActivityLive(await GetActivityStatus()); } catch { /* noop */ }
      } else {
        setActivityLive(null);
      }
    } catch (e) {
      alert(String(e?.message || e));
    } finally {
      setActivityBusy(false);
    }
  }, [activityEnabled]);

  const toggleActivityAlerts = React.useCallback(async () => {
    const next = !activityAlertsOn;
    setActivityAlertsOn(next);
    try { await SetSetting('activity.alerts', next ? '1' : '0'); } catch { /* noop */ }
  }, [activityAlertsOn]);

  const saveActivityThresholds = React.useCallback(async (dailyMin, sessionMin) => {
    setActivityDailyMin(dailyMin);
    setActivitySessionMin(sessionMin);
    try {
      await SetSetting('activity.daily_threshold', String(Math.round(dailyMin * 60)));
      await SetSetting('activity.session_threshold', String(Math.round(sessionMin * 60)));
    } catch { /* noop */ }
  }, []);

  const clearActivityData = React.useCallback(async () => {
    try {
      await ClearActivity('');
      setActivityClearMsg(t.clearedOk);
      setTimeout(() => setActivityClearMsg(''), 2500);
      try { setActivityStats(await GetActivityStats()); } catch { /* noop */ }
    } catch (e) {
      setActivityClearMsg(String(e?.message || e));
    }
  }, [t]);

  const refreshActivityStats = React.useCallback(async () => {
    try { setActivityStats(await GetActivityStats()); } catch { /* noop */ }
  }, []);

  const saveActivityRetention = React.useCallback(async (days) => {
    const d = Math.max(0, Math.min(3650, +days || 0));
    setActivityRetentionDays(d);
    try {
      await SetActivityRetention(d);
      await refreshActivityStats();
    } catch (e) {
      alert(String(e?.message || e));
    }
  }, [refreshActivityStats]);

  const pruneActivityNow = React.useCallback(async () => {
    try {
      const n = await PruneActivityNow();
      setActivityPruneMsg(t.prunedMsg(n ?? 0));
      setTimeout(() => setActivityPruneMsg(''), 3000);
      await refreshActivityStats();
    } catch (e) {
      setActivityPruneMsg(String(e?.message || e));
    }
  }, [t, refreshActivityStats]);

  // ---------- prayer alerts ----------

  const refreshPrayerStatus = React.useCallback(async () => {
    try { setPrayerStatus(await GetPrayerStatus()); } catch { /* noop */ }
  }, []);

  const togglePrayer = React.useCallback(async (next) => {
    const base = prayerSettings || { method: 'egypt', asrHanafi: false, city: 'cairo', lat: 30.0444, lng: 31.2357, tz: 'Africa/Cairo', clock12h: true };
    const updated = { ...base, enabled: !!next };
    setPrayerSettings(updated);
    try {
      await SetPrayerSettings(updated);
      await refreshPrayerStatus();
    } catch (e) {
      alert(String(e?.message || e));
    }
  }, [prayerSettings, refreshPrayerStatus]);

  const savePrayer = React.useCallback(async (draft) => {
    try {
      await SetPrayerSettings({ ...draft, enabled: draft.enabled ?? prayerSettings?.enabled ?? false });
      setPrayerSettings(await GetPrayerSettings());
      await refreshPrayerStatus();
      setPrayerSavedFlash(true);
      setTimeout(() => setPrayerSavedFlash(false), 1500);
    } catch (e) {
      alert(String(e?.message || e));
    }
  }, [prayerSettings, refreshPrayerStatus]);

  const snoozePrayer = React.useCallback(async (minutes) => {
    try { await PrayerSnooze(minutes); } catch { /* noop */ }
    setPrayerDue(null);
  }, []);

  const goingPrayer = React.useCallback(async () => {
    try { await PrayerGoing(); } catch { /* noop */ }
    setPrayerDue(null);
  }, []);

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
      let count = 1;
      try {
        const all = await GetActiveTimers();
        if (Array.isArray(all) && all.length > 0) count = all.length;
      } catch { /* single-timer fallback */ }
      setActiveCount(count);
      setActive({ id: a.id, title: a.title, elapsed: liveElapsed(a, Date.now()), timerStartedAt: a.timerStartedAt, maxSeconds: a.maxSeconds || 0, todaySeconds: a.todaySeconds || 0, totalSeconds: a.totalSeconds ?? a.elapsedSeconds ?? 0 });
    } catch {
      setActive(null);
      setActiveCount(1);
    }
  }, []);

  const refreshContexts = React.useCallback(async () => {
    const day = todayStr();
    try {
      setContexts(await ListContexts(day) || []);
    } catch {
      setContexts([]);
    }
    try {
      const a = await GetActiveContextTimer(day);
      setActiveContext({
        id: a.id, name: a.name, color: a.color,
        elapsed: liveElapsed({ elapsedSeconds: a.elapsedSeconds, timerStartedAt: a.timerStartedAt }, Date.now()),
        elapsedSeconds: a.elapsedSeconds || 0,
        totalSeconds: a.totalSeconds ?? a.elapsedSeconds ?? 0,
        timerStartedAt: a.timerStartedAt, maxSeconds: a.maxSeconds || 0,
        dailyTargetSeconds: a.dailyTargetSeconds || 0, recurrence: a.recurrence || 'daily',
        todaySeconds: a.todaySeconds || 0, weekSeconds: a.weekSeconds || 0,
        tasksTodaySeconds: a.tasksTodaySeconds || 0, tasksTotalSeconds: a.tasksTotalSeconds || 0,
      });
      lastCtxRef.current = { id: a.id, name: a.name, color: a.color };
    } catch {
      setActiveContext(null);
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
    await refreshContexts();
  }, [focusOnly, contextFilter, statusFilter, search, syncActive, refreshContexts, horizons]);

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
        const [hs, cs, db, backendSettings] = await Promise.all([ListHorizons(), ListContexts(todayStr()), GetDBPath(), GetSettings().catch(() => ({}))]);
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
        // backend language wins when saved (Settings → General persists it)
        try {
          const savedLang = (backendSettings?.['app.language'] || '').trim().toLowerCase();
          if (savedLang === 'en' || savedLang === 'ar') {
            setLang(savedLang);
          } else {
            try {
              const bl = await GetAppLanguage();
              if (bl === 'en' || bl === 'ar') setLang(bl);
            } catch { /* keep local */ }
          }
        } catch { /* keep local */ }
        // autostart is OFF by default — reflect the real OS state
        try {
          setAutostartOn(await GetAutostartEnabled());
        } catch { /* unsupported platform */ }
        // activity tracking is OFF by default — reflect persisted flag
        try {
          const en = await GetActivityEnabled();
          setActivityEnabled(!!en);
          const m = backendSettings || {};
          if (m['activity.alerts'] === '0' || m['activity.alerts'] === 'false') setActivityAlertsOn(false);
          const d = parseInt(m['activity.daily_threshold'] || '1800', 10);
          if (!Number.isNaN(d) && d >= 0) setActivityDailyMin(Math.round(d / 60));
          const s = parseInt(m['activity.session_threshold'] || '600', 10);
          if (!Number.isNaN(s) && s >= 0) setActivitySessionMin(Math.round(s / 60));
          try {
            setActivityRetentionDays(await GetActivityRetention());
          } catch { /* keep default */ }
          try {
            setActivityStats(await GetActivityStats());
          } catch { /* noop */ }
          try {
            setPrayerSettings(await GetPrayerSettings());
            setPrayerStatus(await GetPrayerStatus());
          } catch { /* prayer unavailable */ }
          if (en) {
            try { setActivityLive(await GetActivityStatus()); } catch { /* noop */ }
          }
        } catch { /* tracking unavailable */ }
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
        await refreshContexts();
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
  }, [syncActive, refreshContexts]); // eslint-disable-line

  React.useEffect(() => {
    if (ready) refresh().catch((e) => setError(String(e)));
  }, [ready, refresh]);

  // live backend ticks (1s while a timer runs)
  React.useEffect(() => {
    const off = EventsOn('timer:tick', (payload) => {
      if (!payload) {
        setActive(null);
        setActiveCount(1);
        return;
      }
      if (payload.count) setActiveCount(payload.count);
      setActive((prev) => ({
        id: payload.taskId,
        title: payload.title,
        elapsed: payload.elapsed,
        timerStartedAt: prev && prev.id === payload.taskId ? prev.timerStartedAt : prev?.timerStartedAt,
        maxSeconds: prev && prev.id === payload.taskId ? prev.maxSeconds : prev?.maxSeconds,
        todaySeconds: prev && prev.id === payload.taskId ? prev.todaySeconds : prev?.todaySeconds,
        totalSeconds: prev && prev.id === payload.taskId ? prev.totalSeconds : prev?.totalSeconds,
      }));
    });
    const offCtx = EventsOn('context:tick', (payload) => {
      if (!payload) {
        setActiveContext(null);
        return;
      }
      setActiveContext((prev) => ({
        id: payload.contextId,
        name: payload.title,
        color: prev && prev.id === payload.contextId ? prev.color : prev?.color,
        elapsed: payload.elapsed,
        elapsedSeconds: prev && prev.id === payload.contextId ? prev.elapsedSeconds : prev?.elapsedSeconds,
        totalSeconds: prev && prev.id === payload.contextId ? prev.totalSeconds : prev?.totalSeconds,
        timerStartedAt: prev && prev.id === payload.contextId ? prev.timerStartedAt : prev?.timerStartedAt,
        maxSeconds: prev && prev.id === payload.contextId ? prev.maxSeconds : prev?.maxSeconds,
        dailyTargetSeconds: prev && prev.id === payload.contextId ? prev.dailyTargetSeconds : prev?.dailyTargetSeconds,
        recurrence: prev && prev.id === payload.contextId ? prev.recurrence : prev?.recurrence,
        todaySeconds: prev && prev.id === payload.contextId ? prev.todaySeconds : prev?.todaySeconds,
        weekSeconds: prev && prev.id === payload.contextId ? prev.weekSeconds : prev?.weekSeconds,
        tasksTodaySeconds: prev && prev.id === payload.contextId ? prev.tasksTodaySeconds : prev?.tasksTodaySeconds,
        tasksTotalSeconds: prev && prev.id === payload.contextId ? prev.tasksTotalSeconds : prev?.tasksTotalSeconds,
      }));
    });
    return () => { off?.(); offCtx?.(); };
  }, []);

  // app-usage tracking: live status + distraction toasts from the backend
  React.useEffect(() => {
    const offTick = EventsOn('activity:tick', (s) => {
      setActivityLive(s || null);
    });
    const offAlert = EventsOn('activity:distraction', (info) => {
      try {
        const kind = info?.kind === 'daily' ? 'daily' : 'session';
        const app = info?.app || '';
        const secs = kind === 'daily' ? (info?.daySeconds || 0) : (info?.sessionSeconds || 0);
        const hh = formatHMS(secs);
        const msg = kind === 'daily' ? t.distractionDaily(hh) : t.distractionSession(app, hh);
        setDistraction({ kind, app, detail: info?.detail || '', msg });
        setTimeout(() => setDistraction((d) => (d?.msg === msg ? null : d)), 14000);
      } catch { /* noop */ }
    });
    const offEn = EventsOn('activity:enabled', (on) => {
      setActivityEnabled(!!on);
    });
    return () => { offTick?.(); offAlert?.(); offEn?.(); };
  }, [t]);

  // prayer countdown refresh (every minute while enabled) + due modal
  React.useEffect(() => {
    if (!prayerSettings?.enabled) return;
    refreshPrayerStatus();
    const id = setInterval(refreshPrayerStatus, 60000);
    return () => clearInterval(id);
  }, [prayerSettings?.enabled, refreshPrayerStatus]);

  React.useEffect(() => {
    const off = EventsOn('prayer:due', (info) => {
      setPrayerDue(info || null);
    });
    return () => { off?.(); };
  }, []);

  // refresh storage stats whenever the Activity settings tab is opened
  React.useEffect(() => {
    if (settings.open && settings.tab === 'activity') refreshActivityStats();
  }, [settings.open, settings.tab, refreshActivityStats]);

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
        setSheet({ open: true, task: null, presetStatus: 'todo', presetParentId: '' });
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

  // ---------- context-goal actions (parallel timer — independent from task timer) ----------

  const startContext = async (context) => {
    await StartContextTimer(context.id);
    await refreshContexts();
  };
  const pauseContext = async (contextOrId) => {
    const id = typeof contextOrId === 'string' ? contextOrId : contextOrId.id;
    await StopContextTimer(id);
    await refreshContexts();
  };
  const resumeLastContext = async () => {
    if (lastCtxRef.current) await startContext(lastCtxRef.current);
  };

  const MINI_SIZES = {
    widget: { w: 216, h: 54, minW: 200, minH: 48 },
    widget2: { w: 232, h: 92, minW: 210, minH: 80 },
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

  // grow the widget bar when both timers run so each stays visible
  React.useEffect(() => {
    if (miniMode !== 'widget') return;
    applyMiniWindow(active && activeContext ? 'widget2' : 'widget');
  }, [miniMode, active?.id, activeContext?.id]); // eslint-disable-line

  const enterMini = async () => {
    if (!active && !activeContext) return;
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
      try {
        if (activeContext) await StopContextTimer(activeContext.id);
      } catch { /* already stopped */ }
    }
  };

  const openTaskSheet = async (task, presetStatus, presetParentId) => {
    if (task) {
      try {
        setEntries(await ListTimeEntries(task.id));
      } catch {
        setEntries([]);
      }
    } else {
      setEntries([]);
    }
    setSheet({ open: true, task: task || null, presetStatus: presetStatus || 'todo', presetParentId: presetParentId || '' });
  };

  const saveTask = async (input) => {
    if (sheet.task) {
      await UpdateTask(sheet.task.id, input);
      // Parent changes for existing tasks go through the dedicated nest API
      // (UpdateTask preserves the link) so drag cycles stay guarded backend-side.
      if ((input.parentId || '') !== (sheet.task.parentId || '')) {
        try {
          await SetTaskParent(sheet.task.id, input.parentId || '');
        } catch (err) {
          alert(String(err?.message || err));
        }
      }
    } else {
      const created = await CreateTask(input);
      if (created?.horizon) setActiveHorizon(created.horizon);
    }
    setSheet({ open: false, task: null, presetStatus: 'todo', presetParentId: '' });
    await refresh();
  };

  // Unified drag handler: optional parent nest + status move + manual order.
  const handleTaskMove = async (dragId, patch) => {
    try {
      const cur = tasks.find((x) => x.id === dragId);
      if (patch.parentId !== undefined && (cur?.parentId || null) !== (patch.parentId || null)) {
        await SetTaskParent(dragId, patch.parentId || '');
      }
      if (patch.status && cur?.status !== patch.status) {
        await MoveTask(dragId, patch.status);
      }
      if (patch.orderedIds?.length) {
        await ReorderTasks(patch.orderedIds);
      }
    } catch (err) {
      alert(String(err?.message || err));
    }
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
              ctx={activeContext}
              lastCtx={activeContext ? null : lastCtxRef.current}
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
              ctx={activeContext}
              onPauseCtx={() => activeContext && pauseContext(activeContext.id)}
              onResumeCtx={() => activeContext && startContext(activeContext)}
              lastCtx={activeContext ? null : lastCtxRef.current}
              onResumeLastCtx={resumeLastContext}
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
        <img src={brandLogo} alt="Goals" className="w-44 shrink-0 rounded-2xl" />
        <div className="mt-5 flex items-center gap-3 text-sm text-muted-foreground">
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
        timerRunning={!!active || !!activeContext}
        onNewTask={() => openTaskSheet(null, 'todo')}
        onView={setView}
        onHorizon={(h) => { setActiveHorizon(h); setFocusOnly(false); }}
        onFocusToggle={() => setFocusOnly((v) => !v)}
        onClearFilters={clearFilters}
        onMini={enterMini}
        onSettings={(tab) => openSettings(tab)}
        onLangToggle={() => changeLang(lang === 'ar' ? 'en' : 'ar')}
        onOpenInsights={() => setInsightsOpen(true)}
        onQuit={quitApp}
      />
      <div className="flex min-h-0 flex-1">
      <Sidebar
        t={t}
        lang={lang}
        ap={appearance}
        onLangToggle={() => changeLang(lang === 'ar' ? 'en' : 'ar')}
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
        timerRunning={!!active || !!activeContext}
        onQuit={quitApp}
        onSettings={() => openSettings('general')}
        onOpenInsights={() => setInsightsOpen(true)}
        activityEnabled={activityEnabled}
        prayerStatus={prayerStatus}
        onOpenPrayer={() => openSettings('prayer')}
        onOpenFolder={async () => { try { await OpenDataFolder(); } catch { /* noop */ } }}
        version={version}
        updateAvailable={!!latestTag}
        onOpenUpdates={() => openSettings('data')}
        onStartContext={startContext}
        onPauseContext={pauseContext}
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
                activeCount={activeCount}
                ap={appearance}
                onPause={() => pauseTimer(active.id)}
                onResume={() => startTimer(active)}
                onMini={enterMini}
                onFinish={() => askFinish({ id: active.id, title: active.title, elapsedSeconds: active.elapsed, timerStartedAt: active.timerStartedAt })}
              />
            )}
            {activeContext && (
              <ActiveContextPill
                t={t}
                active={activeContext}
                ap={appearance}
                onPause={() => pauseContext(activeContext.id)}
                onResume={() => startContext(activeContext)}
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
              onTaskMove={handleTaskMove}
              onAddSubtask={(parent) => openTaskSheet(null, parent.status, parent.id)}
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
                onTaskMove={handleTaskMove}
                onAddSubtask={(parent) => openTaskSheet(null, parent.status, parent.id)}
              />
            </div>
          )}
        </div>
      </main>
      </div>

      <TaskSheet
        t={t}
        open={sheet.open}
        onClose={() => setSheet({ open: false, task: null, presetStatus: 'todo', presetParentId: '' })}
        task={sheet.task}
        horizons={horizons}
        contexts={contexts}
        activeHorizon={activeHorizon}
        presetStatus={sheet.presetStatus}
        presetParentId={sheet.presetParentId}
        tasks={tasks}
        onSave={saveTask}
        entries={entries}
      />
      <SettingsSheet
        t={t}
        lang={lang}
        onLangChange={changeLang}
        autostartOn={autostartOn}
        autostartBusy={autostartBusy}
        autostartErr={autostartErr}
        onAutostartToggle={toggleAutostart}
        activityEnabled={activityEnabled}
        activityBusy={activityBusy}
        onToggleActivity={toggleActivity}
        activityAlertsOn={activityAlertsOn}
        onToggleActivityAlerts={toggleActivityAlerts}
        activityDailyMin={activityDailyMin}
        activitySessionMin={activitySessionMin}
        onSaveActivityThresholds={saveActivityThresholds}
        activityLive={activityLive}
        onOpenInsights={() => { setSettings((s) => ({ ...s, open: false })); setInsightsOpen(true); }}
        onClearActivity={clearActivityData}
        activityClearMsg={activityClearMsg}
        activityRetentionDays={activityRetentionDays}
        onSaveActivityRetention={saveActivityRetention}
        activityStats={activityStats}
        onPruneActivityNow={pruneActivityNow}
        activityPruneMsg={activityPruneMsg}
        prayerSettings={prayerSettings}
        onTogglePrayer={togglePrayer}
        onSavePrayer={savePrayer}
        prayerSavedFlash={prayerSavedFlash}
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
        onCreateContext={async (n, c) => { await CreateContext(n, c); await refreshContexts(); }}
        onUpdateContexts={async (drafts) => {
          for (const d of drafts) {
            const orig = contexts.find((x) => x.id === d.id);
            if (orig && (orig.name !== d.name || orig.color !== d.color
              || (orig.dailyTargetSeconds || 0) !== (d.dailyTargetSeconds || 0)
              || (orig.maxSeconds || 0) !== (d.maxSeconds || 0)
              || (orig.recurrence || 'daily') !== (d.recurrence || 'daily')
              || (orig.description || '') !== (d.description || '')
              || (orig.descriptionAr || '') !== (d.descriptionAr || ''))) {
              await UpdateContext(d.id, d.name, d.color, d.dailyTargetSeconds || 0, d.maxSeconds || 0, d.recurrence || 'daily', d.description || '', d.descriptionAr || '');
            }
          }
          await refresh();
        }}
        onDeleteContext={async (d) => {
          await DeleteContext(d.id);
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
      <ActivityInsights
        t={t}
        open={insightsOpen}
        onClose={() => setInsightsOpen(false)}
        enabled={activityEnabled}
      />
      <PrayerAlert
        t={t}
        lang={lang}
        event={prayerDue}
        use12h={prayerSettings?.clock12h !== false}
        onSnooze={snoozePrayer}
        onGoing={goingPrayer}
        onClose={() => setPrayerDue(null)}
      />
      {distraction && (
        <div className="fixed bottom-5 start-5 z-50 max-w-sm rounded-xl border border-amber-500/40 bg-card p-3.5 shadow-2xl animate-slide-in">
          <div className="flex items-start gap-2.5">
            <div className="min-w-0 flex-1">
              <p className="text-[13px] font-bold text-amber-300">
                {distraction.kind === 'daily' ? t.wastedTime : distraction.app || t.wastedTime}
              </p>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{distraction.msg}</p>
              <div className="mt-2 flex gap-2">
                <button
                  onClick={() => { setDistraction(null); setInsightsOpen(true); }}
                  className="rounded-md bg-primary px-2.5 py-1 text-[11px] font-bold text-primary-foreground hover:opacity-90"
                >
                  {t.openInsights}
                </button>
                <button
                  onClick={() => setDistraction(null)}
                  className="rounded-md border px-2.5 py-1 text-[11px] font-bold text-muted-foreground hover:text-foreground"
                >
                  {t.dismiss}
                </button>
              </div>
            </div>
            <button onClick={() => setDistraction(null)} className="text-muted-foreground hover:text-foreground">
              <X size={14} />
            </button>
          </div>
        </div>
      )}
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
