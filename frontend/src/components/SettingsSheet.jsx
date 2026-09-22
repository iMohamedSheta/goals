import * as React from 'react';
import {
  Moon, Sun, Monitor, Palette, Type, Square, Layers, Box,
  Sparkles, Maximize2, RotateCcw, Plus, Trash2, CalendarRange, Users,
  Database, Cloud, Copy, Check, FolderOpen, Upload, Download, Bot, Power,
  Globe, Activity, BarChart3, Languages, Sunrise, MapPin, BellRing,
} from 'lucide-react';
import { cn } from '../lib/utils';
import { Button } from './ui/button';
import { Sheet, SheetHeader, SheetBody, SheetFooter } from './ui/sheet';
import { Input, Label, Select } from './ui/form';
import { ACCENTS, FONTS, surfClass, density, motionClass, RADIUS } from '../lib/appearance';
import { horizonName, formatHMS, formatClock } from '../lib/i18n';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { ListPrayerCities, GetPrayerTimesFor } from '../../wailsjs/go/main/App';

function Seg({ options, value, onPick }) {
  return (
    <div className="inline-flex flex-wrap gap-1 rounded-xl bg-muted p-1">
      {options.map((o) => (
        <button
          key={o.value}
          onClick={() => onPick(o.value)}
          className={cn(
            'flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-bold transition-all',
            value === o.value ? 'border bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
          )}
        >
          {o.Icon && <o.Icon size={14} />}
          {o.label}
        </button>
      ))}
    </div>
  );
}

function Section({ icon: Icon, title, children }) {
  return (
    <div className="rounded-xl border p-3.5">
      <div className="mb-2.5 flex items-center gap-2 text-[13px] font-bold">
        <Icon size={15} className="text-primary" />
        {title}
      </div>
      {children}
    </div>
  );
}

function Switch({ on, onToggle }) {
  return (
    <button
      onClick={onToggle}
      className={cn('relative h-6 w-11 shrink-0 rounded-full transition-colors', on ? 'bg-primary' : 'bg-input')}
    >
      <span className={cn('absolute top-0.5 size-5 rounded-full bg-white shadow transition-all', on ? 'end-0.5' : 'start-0.5')} />
    </button>
  );
}

function Field({ label, children }) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      {children}
    </div>
  );
}

/* ---------------- appearance tab ---------------- */

function AppearanceTab({ t, value, dirty, onChange, onReset }) {
  return (
    <div className="space-y-3">
      <div className="rounded-xl border bg-muted/20 p-3">
        <div className="mb-2.5 flex items-center justify-between">
          <span className="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">{t.livePreview}</span>
          {dirty && (
            <button onClick={onReset} className="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-bold text-muted-foreground hover:bg-accent hover:text-foreground">
              <RotateCcw size={12} /> {t.resetDefault}
            </button>
          )}
        </div>
        <div className={cn('tcard', RADIUS, surfClass(value), density(value).card, motionClass(value))}>
          <div className="h-2 w-12 rounded-full bg-primary/25" />
          <div className="mt-2 h-2.5 w-full rounded bg-foreground/10" />
          <div className="mt-1.5 h-2.5 w-2/3 rounded bg-foreground/5" />
          <div className="mt-2.5 flex gap-1.5">
            <span className="grid h-7 flex-1 place-items-center rounded-md bg-primary text-[11px] font-bold text-primary-foreground">{t.newTask}</span>
            <span className="grid h-7 flex-1 place-items-center rounded-md border bg-card text-[11px]">{t.focus}</span>
          </div>
        </div>
        <p className="mt-2 text-center text-xs text-muted-foreground" style={{ fontFamily: FONTS[value.font] }}>
          Aa {value.font} — {t.typography}
        </p>
      </div>

      <Section icon={Monitor} title={t.appTheme}>
        <Seg value={value.theme} onPick={(v) => onChange({ theme: v })} options={[
          { value: 'dark', label: t.themeDark, Icon: Moon },
          { value: 'light', label: t.themeLight, Icon: Sun },
          { value: 'system', label: t.themeSystem, Icon: Monitor },
        ]} />
      </Section>

      <Section icon={Palette} title={t.accentColor}>
        <div className="flex flex-wrap gap-2.5">
          {Object.entries(ACCENTS).map(([name, hsl]) => (
            <button key={name} title={name} onClick={() => onChange({ accent: name })}
              className={cn('size-9 rounded-full border-2 transition-all', value.accent === name ? 'scale-110 border-foreground' : 'border-white/20 hover:scale-105')}
              style={{ background: `hsl(${hsl})` }} />
          ))}
        </div>
      </Section>

      <Section icon={Type} title={t.typography}>
        <div className="mb-3 flex flex-wrap gap-1.5">
          {['Cairo', 'Tajawal', 'system'].map((f) => (
            <button key={f} onClick={() => onChange({ font: f })} style={{ fontFamily: FONTS[f] }}
              className={cn(RADIUS, 'border px-3.5 py-1.5 text-[13px] transition-all',
                value.font === f ? 'border-primary bg-primary font-bold text-primary-foreground shadow-sm' : 'border-border bg-card text-muted-foreground hover:border-primary/30 hover:text-foreground')}>
              {f === 'system' ? t.fontSystem : f}
            </button>
          ))}
        </div>
        <Label className="mb-1.5 block text-[11px] font-bold uppercase tracking-wider text-muted-foreground">{t.fontSize}</Label>
        <Seg value={value.fontSize} onPick={(v) => onChange({ fontSize: v })} options={[
          { value: 'sm', label: t.sizeSmall },
          { value: 'md', label: t.sizeNormal },
          { value: 'lg', label: t.sizeLarge },
          { value: 'xl', label: t.sizeXL },
        ]} />
      </Section>

      <Section icon={Square} title={t.radiusLabel}>
        <div className="grid grid-cols-5 gap-2">
          {[
            { value: 'none', label: t.radiusNone, r: '2px' },
            { value: 'sharp', label: t.radiusSharp, r: '5px' },
            { value: 'balanced', label: t.radiusBalanced, r: '10px' },
            { value: 'soft', label: t.radiusSoft, r: '16px' },
            { value: 'round', label: t.radiusRound, r: '22px' },
          ].map((o) => (
            <button key={o.value} onClick={() => onChange({ radius: o.value })}
              className={cn('flex flex-col items-center gap-1.5 rounded-xl border-2 p-2 transition-all',
                value.radius === o.value ? 'border-primary bg-primary/5' : 'border-border bg-card hover:border-primary/30')}>
              <span className="h-7 w-full border bg-muted/40" style={{ borderRadius: o.r }} />
              <span className={cn('text-[10px] font-bold', value.radius === o.value ? 'text-primary' : 'text-muted-foreground')}>{o.label}</span>
            </button>
          ))}
        </div>
      </Section>

      <Section icon={Layers} title={t.cardStyleLabel}>
        <div className="grid grid-cols-4 gap-2">
          {[
            { value: 'elevated', label: t.cardElevated, Icon: Layers },
            { value: 'bordered', label: t.cardBordered, Icon: Box },
            { value: 'flat', label: t.cardFlat, Icon: Square },
            { value: 'glass', label: t.cardGlass, Icon: Sparkles },
          ].map((o) => (
            <button key={o.value} onClick={() => onChange({ cardStyle: o.value })}
              className={cn('flex flex-col items-center gap-1.5 rounded-xl border-2 p-2 transition-all',
                value.cardStyle === o.value ? 'border-primary bg-primary/5' : 'border-border bg-card hover:border-primary/30')}>
              <o.Icon size={17} className={value.cardStyle === o.value ? 'text-primary' : 'text-muted-foreground'} />
              <span className={cn('text-[10px] font-bold', value.cardStyle === o.value ? 'text-primary' : 'text-muted-foreground')}>{o.label}</span>
            </button>
          ))}
        </div>
      </Section>

      <Section icon={Maximize2} title={t.densityLabel}>
        <Seg value={value.density} onPick={(v) => onChange({ density: v })} options={[
          { value: 'comfortable', label: t.densityComfort },
          { value: 'compact', label: t.densityCompact },
          { value: 'spacious', label: t.densitySpacious },
        ]} />
      </Section>

      <Section icon={Sparkles} title={t.animationsLabel}>
        <div className="space-y-2.5">
          <div className="flex items-center justify-between gap-3">
            <p className="text-[13px] font-bold">{t.shadowsLabel}</p>
            <Switch on={value.shadows === 'on'} onToggle={() => onChange({ shadows: value.shadows === 'on' ? 'off' : 'on' })} />
          </div>
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-[13px] font-bold">{t.animationsLabel}</p>
              <p className="text-[11px] text-muted-foreground">{t.animationsDesc}</p>
            </div>
            <Switch on={value.animations === 'on'} onToggle={() => onChange({ animations: value.animations === 'on' ? 'off' : 'on' })} />
          </div>
          <div className="flex items-center justify-between gap-3">
            <p className="text-[13px] font-bold">{t.glassSidebar}</p>
            <Switch on={value.glass === 'on'} onToggle={() => onChange({ glass: value.glass === 'on' ? 'off' : 'on' })} />
          </div>
          <div className="flex items-center justify-between gap-3">
            <p className="text-[13px] font-bold">{t.cardDescLabel}</p>
            <Switch on={value.cardDesc === 'on'} onToggle={() => onChange({ cardDesc: value.cardDesc === 'on' ? 'off' : 'on' })} />
          </div>
        </div>
      </Section>
    </div>
  );
}

/* ---------------- general tab (language + OS startup) ---------------- */

function GeneralTab({ t, lang, onLangChange, autostartOn, autostartBusy, autostartErr, onAutostartToggle }) {
  return (
    <div className="space-y-3">
      <Section icon={Languages} title={t.languageLabel}>
        <p className="mb-2.5 text-xs leading-relaxed text-muted-foreground">{t.languageDesc}</p>
        <div className="inline-flex flex-wrap gap-1 rounded-xl bg-muted p-1">
          {[
            { value: 'ar', label: t.langArabic },
            { value: 'en', label: t.langEnglish },
          ].map((o) => (
            <button
              key={o.value}
              onClick={() => onLangChange(o.value)}
              className={cn(
                'rounded-lg px-4 py-1.5 text-xs font-bold transition-all',
                lang === o.value ? 'border bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {o.label}
            </button>
          ))}
        </div>
      </Section>

      <Section icon={Power} title={t.autostartLabel}>
        <div className="flex items-center justify-between gap-3">
          <p className="text-xs leading-relaxed text-muted-foreground">{t.autostartDesc}</p>
          <Switch on={!!autostartOn} onToggle={onAutostartToggle} />
        </div>
        {autostartBusy && <p className="mt-2 text-xs text-muted-foreground">…</p>}
        {autostartErr && <p className="mt-2 text-xs text-red-400" dir="ltr">{autostartErr}</p>}
      </Section>
    </div>
  );
}

/* ---------------- activity tab (opt-in usage tracking) ---------------- */

function ActivityTab({ t, enabled, busy, onToggleEnabled, alertsOn, onToggleAlerts, dailyMin, sessionMin, onSaveThresholds, live, onOpenInsights, onClear, clearMsg, retentionDays, onSaveRetention, stats, onPruneNow, pruneMsg }) {
  const [daily, setDaily] = React.useState(dailyMin ?? 30);
  const [session, setSession] = React.useState(sessionMin ?? 10);
  const [saved, setSaved] = React.useState(false);
  const [keep, setKeep] = React.useState(retentionDays ?? 90);
  const [keepSaved, setKeepSaved] = React.useState(false);
  React.useEffect(() => { setDaily(dailyMin ?? 30); }, [dailyMin]);
  React.useEffect(() => { setSession(sessionMin ?? 10); }, [sessionMin]);
  React.useEffect(() => { setKeep(retentionDays ?? 90); }, [retentionDays]);

  const save = async () => {
    setSaved(false);
    await onSaveThresholds?.(Math.max(0, +daily || 0), Math.max(0, +session || 0));
    setSaved(true);
    setTimeout(() => setSaved(false), 1500);
  };

  const saveKeep = async () => {
    setKeepSaved(false);
    await onSaveRetention?.(Math.max(0, Math.min(3650, +keep || 0)));
    setKeepSaved(true);
    setTimeout(() => setKeepSaved(false), 1500);
  };

  const fmtBytes = (n) => {
    n = n || 0;
    if (n < 1024) return `${n} B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
    if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
    return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
  };

  return (
    <div className="space-y-3">
      <Section icon={Activity} title={t.activityTitle}>
        <p className="mb-2.5 text-xs leading-relaxed text-muted-foreground">{t.activityDesc}</p>
        <div className="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-muted/20 p-2.5">
          <div>
            <p className="text-[13px] font-bold">{t.activityEnabledLabel}</p>
            <p className="mt-0.5 text-[11px] leading-relaxed text-muted-foreground">{t.activityEnabledDesc}</p>
          </div>
          <Switch on={!!enabled} onToggle={onToggleEnabled} />
        </div>
        {busy && <p className="mt-2 text-xs text-muted-foreground">…</p>}
        {live?.enabled && live?.app ? (
          <p className="tabular mt-2 text-[11px] text-muted-foreground">
            {t.liveNow}: <span className="font-bold text-foreground">{live.detail || live.title || live.app}</span>
            {' · '}{live.app}{live.domain ? ` · ${live.domain}` : ''}
          </p>
        ) : null}
        <p className="mt-2 text-[11px] leading-relaxed text-muted-foreground">{t.activityPrivacy}</p>
        <p className="mt-1 text-[11px] leading-relaxed text-muted-foreground">{t.activityHow}</p>
      </Section>

      <Section icon={BarChart3} title={t.insights}>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" onClick={onOpenInsights}>
            <BarChart3 /> {t.openInsights}
          </Button>
          <Button variant="secondary" size="sm" onClick={onClear}>
            <Trash2 /> {t.clearActivity}
          </Button>
        </div>
        {clearMsg && <p className="mt-2 text-xs text-muted-foreground">{clearMsg}</p>}
      </Section>

      <Section icon={Database} title={t.dbSize}>
        <p className="tabular text-xs text-muted-foreground">
          {(stats?.segments ?? 0)} {t.sessionsLabel} · {fmtBytes(stats?.dbBytes)} {stats?.oldest ? `· ${t.storedSince(stats.oldest)}` : ''}
        </p>
        <p className="mt-2 text-xs leading-relaxed text-muted-foreground">{t.retentionDesc}</p>
        <div className="mt-2.5 flex items-end gap-2">
          <div className="w-36">
            <Field label={t.retentionLabel}>
              <Input type="number" min={0} max={3650} value={keep} onChange={(e) => setKeep(e.target.value)} className="tabular text-center" />
            </Field>
          </div>
          <Button size="sm" onClick={saveKeep}>
            {keepSaved ? <Check /> : null} {keepSaved ? t.copied : t.saveChanges}
          </Button>
          <Button variant="secondary" size="sm" onClick={onPruneNow}>
            <Trash2 /> {t.pruneNow}
          </Button>
        </div>
        {pruneMsg && <p className="mt-2 text-xs text-muted-foreground">{pruneMsg}</p>}
      </Section>

      <Section icon={Power} title={t.activityAlertsLabel}>
        <div className="mb-2.5 flex items-center justify-between gap-3">
          <p className="text-xs leading-relaxed text-muted-foreground">{t.activityAlertsDesc}</p>
          <Switch on={!!alertsOn} onToggle={onToggleAlerts} />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <Field label={t.activityDailyLabel}>
            <Input type="number" min={0} max={1440} value={daily} onChange={(e) => setDaily(e.target.value)} className="tabular text-center" />
          </Field>
          <Field label={t.activitySessionLabel}>
            <Input type="number" min={0} max={480} value={session} onChange={(e) => setSession(e.target.value)} className="tabular text-center" />
          </Field>
        </div>
        <p className="mt-2 text-[11px] text-muted-foreground">{t.activityThresholdHint}</p>
        <Button size="sm" className="mt-2.5" onClick={save}>
          {saved ? <Check /> : null} {saved ? t.copied : t.saveChanges}
        </Button>
      </Section>
    </div>
  );
}

/* ---------------- prayer tab ---------------- */

const PRAYER_METHODS = [
  { value: 'egypt', labelKey: 'methodEgypt' },
  { value: 'mwl', labelKey: 'methodMwl' },
  { value: 'isna', labelKey: 'methodIsna' },
  { value: 'makkah', labelKey: 'methodMakkah' },
  { value: 'karachi', labelKey: 'methodKarachi' },
  { value: 'gulf', labelKey: 'methodGulf' },
  { value: 'jafari', labelKey: 'methodJafari' },
];

const PRAYER_KEYS = ['fajr', 'sunrise', 'dhuhr', 'asr', 'maghrib', 'isha'];

function PrayerTab({ t, lang, initial, onToggleEnabled, onSave, savedFlash }) {
  const [cities, setCities] = React.useState([]);
  const [custom, setCustom] = React.useState(initial?.city === 'custom');
  const [draft, setDraft] = React.useState(() => ({ clock12h: true, ...(initial || {}) }));
  const [times, setTimes] = React.useState([]);

  React.useEffect(() => {
    ListPrayerCities().then((c) => setCities(c || [])).catch(() => {});
  }, []);
  React.useEffect(() => {
    setDraft({ clock12h: true, ...(initial || {}) });
    setCustom(initial?.city === 'custom');
  }, [initial?.city, initial?.method, initial?.asrHanafi, initial?.lat, initial?.lng, initial?.tz, initial?.clock12h]); // eslint-disable-line

  const set = (patch) => setDraft((d) => ({ ...d, ...patch }));

  // countries in list order (Egypt first), then the places inside one country
  const countries = React.useMemo(() => {
    const seen = [];
    for (const c of cities || []) {
      if (c.country && !seen.includes(c.country)) seen.push(c.country);
    }
    return seen;
  }, [cities]);
  const cityOf = (id) => (cities || []).find((c) => c.id === id);
  const draftCountry = cityOf(draft.city)?.country || countries[0] || 'Egypt';
  const places = (cities || []).filter((c) => c.country === draftCountry);

  const pickCountry = (country) => {
    const first = (cities || []).find((c) => c.country === country);
    if (first) {
      set({ city: first.id, lat: first.lat, lng: first.lng, tz: first.tz });
      setCustom(false);
    }
  };
  const pickCity = (id) => {
    const c = cityOf(id);
    if (c) set({ city: c.id, lat: c.lat, lng: c.lng, tz: c.tz });
  };

  // live preview for the current draft (unsaved) location+method
  React.useEffect(() => {
    const lat = +draft.lat, lng = +draft.lng;
    if (!Number.isFinite(lat) || !Number.isFinite(lng)) { setTimes([]); return; }
    const id = setTimeout(() => {
      GetPrayerTimesFor(lat, lng, draft.tz || 'Africa/Cairo', draft.method || 'egypt', !!draft.asrHanafi, '')
        .then((rows) => setTimes(rows || [])).catch(() => {});
    }, 300);
    return () => clearTimeout(id);
  }, [draft.lat, draft.lng, draft.tz, draft.method, draft.asrHanafi]);

  return (
    <div className="space-y-3">
      <Section icon={BellRing} title={t.prayerTitle}>
        <p className="mb-2.5 text-xs leading-relaxed text-muted-foreground">{t.prayerDesc}</p>
        <div className="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-muted/20 p-2.5">
          <div>
            <p className="text-[13px] font-bold">{t.prayerEnabledLabel}</p>
            <p className="mt-0.5 text-[11px] leading-relaxed text-muted-foreground">{t.prayerEnabledDesc}</p>
          </div>
          <Switch on={!!draft.enabled} onToggle={() => { const next = !draft.enabled; set({ enabled: next }); onToggleEnabled?.(next); }} />
        </div>
      </Section>

      <Section icon={MapPin} title={t.prayerLocation}>
        {!custom ? (
          <>
            <div className="grid grid-cols-2 gap-2.5">
              <Field label={t.countryLabel}>
                <Select value={draftCountry} onChange={(e) => pickCountry(e.target.value)}>
                  {countries.map((c) => (
                    <option key={c} value={c}>{c}</option>
                  ))}
                </Select>
              </Field>
              <Field label={t.placeLabel}>
                <Select value={draft.city || ''} onChange={(e) => pickCity(e.target.value)}>
                  {places.map((c) => (
                    <option key={c.id} value={c.id}>
                      {lang === 'ar' ? (c.nameAr || c.name) : c.name}
                    </option>
                  ))}
                </Select>
              </Field>
            </div>
            {(() => {
              const sel = cityOf(draft.city);
              return sel ? (
                <p className="mt-2 text-[11px] text-muted-foreground" dir="ltr">
                  {sel.lat.toFixed(3)}, {sel.lng.toFixed(3)} · {sel.tz}
                </p>
              ) : null;
            })()}
          </>
        ) : (
          <div className="grid grid-cols-3 gap-2.5">
            <Field label={t.latLabel}>
              <Input type="number" step="any" value={draft.lat ?? ''} onChange={(e) => set({ lat: parseFloat(e.target.value) })} dir="ltr" className="tabular text-center" />
            </Field>
            <Field label={t.lngLabel}>
              <Input type="number" step="any" value={draft.lng ?? ''} onChange={(e) => set({ lng: parseFloat(e.target.value) })} dir="ltr" className="tabular text-center" />
            </Field>
            <Field label={t.tzLabel}>
              <Input value={draft.tz || ''} onChange={(e) => set({ tz: e.target.value })} dir="ltr" placeholder="Africa/Cairo" className="text-center" />
            </Field>
          </div>
        )}
        <button
          onClick={() => { const next = !custom; setCustom(next); if (next) set({ city: 'custom' }); }}
          className="mt-2.5 text-[11px] font-bold text-primary hover:underline"
        >
          {custom ? t.placeLabel : `${t.useCustom} · ${t.customLocation}`}
        </button>
        {custom && <p className="mt-1 text-[11px] text-muted-foreground" dir="ltr">{t.tzHint}</p>}
      </Section>

      <Section icon={Sunrise} title={t.prayerMethod}>
        <Select value={draft.method || 'egypt'} onChange={(e) => set({ method: e.target.value })}>
          {PRAYER_METHODS.map((m) => (
            <option key={m.value} value={m.value}>{t[m.labelKey] || m.value}</option>
          ))}
        </Select>
        <div className="mt-2.5">
          <Label className="mb-1.5 block text-[11px] font-bold uppercase tracking-wider text-muted-foreground">{t.asrLabel}</Label>
          <div className="inline-flex flex-wrap gap-1 rounded-xl bg-muted p-1">
            {[
              { value: false, label: t.asrStandard },
              { value: true, label: t.asrHanafi },
            ].map((o) => (
              <button
                key={String(o.value)}
                onClick={() => set({ asrHanafi: o.value })}
                className={cn(
                  'rounded-lg px-3 py-1.5 text-xs font-bold transition-all',
                  !!draft.asrHanafi === o.value ? 'border bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
                )}
              >
                {o.label}
              </button>
            ))}
          </div>
        </div>
        <div className="mt-2.5">
          <Label className="mb-1.5 block text-[11px] font-bold uppercase tracking-wider text-muted-foreground">{t.clockLabel}</Label>
          <Seg value={draft.clock12h === false ? 'h24' : 'h12'} onPick={(v) => set({ clock12h: v === 'h12' })} options={[
            { value: 'h12', label: t.clock12 },
            { value: 'h24', label: t.clock24 },
          ]} />
        </div>
      </Section>

      <Section icon={Sunrise} title={t.todayTimes}>
        {times.length === 0 ? (
          <p className="py-1 text-xs text-muted-foreground">…</p>
        ) : (
          <div className="grid grid-cols-2 gap-1.5">
            {PRAYER_KEYS.map((k) => {
              const row = times.find((x) => x.key === k);
              return (
                <div key={k} className="flex items-center justify-between rounded-lg border border-border/50 bg-muted/20 px-2.5 py-1.5">
                  <span className="text-xs font-bold">{t[k] || k}</span>
                  <span dir="ltr" className="tabular text-xs text-muted-foreground">{formatClock(row?.time || '', draft.clock12h !== false, lang) || '—'}</span>
                </div>
              );
            })}
          </div>
        )}
        <Button
          size="sm" className="mt-2.5"
          onClick={() => onSave?.({ ...draft, city: custom ? 'custom' : (draft.city || 'cairo') })}
        >
          {savedFlash ? <Check /> : null} {savedFlash ? t.copied : t.saveChanges}
        </Button>
      </Section>
    </div>
  );
}

/* ---------------- planning tab ---------------- */

const BUILTINS = new Set(['short', 'medium', 'long']);

function PlanningTab({ t, lang, horizons, onSave, onCreate, onDelete }) {
  const [rows, setRows] = React.useState([]);
  const [draft, setDraft] = React.useState({ label: '', labelAr: '', days: 30 });
  const [err, setErr] = React.useState('');

  React.useEffect(() => {
    setRows(horizons.map((h) => ({ ...h })));
    setErr('');
  }, [horizons]);

  const set = (key, field, v) => {
    setErr('');
    setRows((rs) => rs.map((r) => (r.key === key ? { ...r, [field]: v } : r)));
  };

  const valid = draft.label.trim() || draft.labelAr.trim();

  return (
    <div className="space-y-3">
      {err && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3.5 py-2.5 text-[13px] text-red-300">{err}</div>
      )}
      {rows.map((r) => (
        <div key={r.key} className="rounded-xl border p-3.5">
          <div className="mb-2.5 flex items-center gap-2">
            <span className="text-sm font-bold">{horizonName(r, lang)}</span>
            <span className={cn('tabular bg-primary/10 px-2 py-0.5 text-[11px] font-semibold text-primary', RADIUS)}>~{r.defaultDays}</span>
            {!BUILTINS.has(r.key) && (
              <span className={cn('bg-secondary px-2 py-0.5 text-[10px] font-bold text-muted-foreground', RADIUS)}>{t.customTab}</span>
            )}
            <span className="ms-auto" />
            <button
              onClick={async () => {
                const res = await onDelete(r);
                if (res && !res.ok) setErr(res.error);
              }}
              title={t.delete}
              className="rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-red-400"
            >
              <Trash2 size={14} />
            </button>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Field label={t.labelEn}>
              <Input value={r.label} onChange={(e) => set(r.key, 'label', e.target.value)} dir="ltr" />
            </Field>
            <Field label={t.labelAr}>
              <Input value={r.labelAr || ''} onChange={(e) => set(r.key, 'labelAr', e.target.value)} />
            </Field>
          </div>
          <div className="mt-3">
            <Field label={t.daysLabel}>
              <Input type="number" min={1} max={3650} value={r.defaultDays} onChange={(e) => set(r.key, 'defaultDays', parseInt(e.target.value, 10) || 1)} />
            </Field>
          </div>
          <div className="mt-3 grid grid-cols-1 gap-3">
            <Field label={t.descEn}>
              <Input value={r.description} onChange={(e) => set(r.key, 'description', e.target.value)} dir="ltr" />
            </Field>
            <Field label={t.descAr}>
              <Input value={r.descriptionAr || ''} onChange={(e) => set(r.key, 'descriptionAr', e.target.value)} />
            </Field>
          </div>
        </div>
      ))}

      <div className="rounded-xl border border-dashed p-3.5">
        <p className="mb-2.5 text-[13px] font-bold">{t.addHorizon}</p>
        <div className="grid grid-cols-2 gap-3">
          <Field label={t.labelEn}>
            <Input value={draft.label} onChange={(e) => setDraft((d) => ({ ...d, label: e.target.value }))} dir="ltr" placeholder="Quarterly" />
          </Field>
          <Field label={t.labelAr}>
            <Input value={draft.labelAr} onChange={(e) => setDraft((d) => ({ ...d, labelAr: e.target.value }))} placeholder="ربع سنوية" />
          </Field>
        </div>
        <div className="mt-3 flex items-end gap-2.5">
          <div className="w-32">
            <Field label={t.daysLabel}>
              <Input type="number" min={1} max={3650} value={draft.days} onChange={(e) => setDraft((d) => ({ ...d, days: parseInt(e.target.value, 10) || 30 }))} />
            </Field>
          </div>
          <Button
            disabled={!valid}
            onClick={async () => {
              const res = await onCreate(draft);
              if (res && !res.ok) setErr(res.error);
              else setDraft({ label: '', labelAr: '', days: 30 });
            }}
          >
            <Plus /> {t.add}
          </Button>
        </div>
        <p className="mt-2 text-[11px] text-muted-foreground">{t.newHorizonHint}</p>
      </div>

      <Button className="w-full" onClick={() => onSave(rows)}>{t.saveTimelines}</Button>
    </div>
  );
}

/* ---------------- contexts tab ---------------- */

function ContextsTab({ t, contexts, onCreate, onUpdate, onDelete }) {
  const [drafts, setDrafts] = React.useState([]);
  const [name, setName] = React.useState('');
  const [color, setColor] = React.useState('#6366f1');

  React.useEffect(() => {
    setDrafts(contexts.map((c) => ({ ...c })));
    setName('');
  }, [contexts]);

  const set = (id, field, v) => setDrafts((ds) => ds.map((d) => (d.id === id ? { ...d, [field]: v } : d)));
  const setSecs = (id, field, h, m) => set(id, field, (h || 0) * 3600 + (m || 0) * 60);
  const hm = (secs) => ({ h: Math.floor((secs || 0) / 3600), m: Math.floor(((secs || 0) % 3600) / 60) });

  return (
    <div className="space-y-2.5">
      {drafts.map((d) => {
        const daily = hm(d.dailyTargetSeconds);
        const max = hm(d.maxSeconds);
        const weekly = (d.recurrence || 'daily') === 'weekly';
        const periodSecs = weekly ? (d.weekSeconds || 0) : (d.todaySeconds || 0);
        const totalOwn = d.totalSeconds ?? d.elapsedSeconds ?? 0;
        return (
          <div key={d.id} className="space-y-3 rounded-xl border bg-card/40 p-3.5">
            <div className="flex items-center gap-2.5">
              <input type="color" value={d.color} onChange={(e) => set(d.id, 'color', e.target.value)}
                className="size-10 shrink-0 cursor-pointer rounded-lg border border-input bg-background p-1.5" />
              <Input
                value={d.name} onChange={(e) => set(d.id, 'name', e.target.value)}
                className="h-10 min-w-0 flex-1 text-sm font-semibold"
              />
              <Button variant="ghost" size="icon-sm" className="h-10 w-10 shrink-0 hover:bg-destructive/10 hover:text-red-400" onClick={() => onDelete(d)}>
                <Trash2 size={15} />
              </Button>
            </div>
            <p className="tabular text-[11px] leading-relaxed text-muted-foreground">
              {weekly ? t.weekLabel : (t.todayTime || t.todayLabel)}: {formatHMS(periodSecs)}
              {' · '}{t.totalLabel || 'Total'}: {formatHMS(totalOwn)}
            </p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <Field label={t.descEn}>
                <Input value={d.description || ''} onChange={(e) => set(d.id, 'description', e.target.value)} dir="ltr" placeholder="e.g. Deep work & clients" />
              </Field>
              <Field label={t.descAr}>
                <Input value={d.descriptionAr || ''} onChange={(e) => set(d.id, 'descriptionAr', e.target.value)} placeholder="مثال: عمل عميق وعملاء" />
              </Field>
            </div>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="rounded-lg border border-border/60 bg-muted/20 p-2.5">
                <div className="mb-2 flex items-center gap-2">
                  <Label>{weekly ? t.weeklyTarget : t.dailyTarget}</Label>
                  <Select
                    value={d.recurrence || 'daily'}
                    onChange={(e) => set(d.id, 'recurrence', e.target.value)}
                    className="ms-auto h-8 w-28 py-1 text-xs font-semibold"
                  >
                    <option value="daily">{t.recDaily}</option>
                    <option value="weekly">{t.recWeekly}</option>
                  </Select>
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Input type="number" min={0} max={999} value={daily.h}
                    onChange={(e) => setSecs(d.id, 'dailyTargetSeconds', Math.max(0, parseInt(e.target.value, 10) || 0), daily.m)}
                    className="h-9 w-20 text-center tabular" />
                  <span>{t.hoursShort}</span>
                  <Input type="number" min={0} max={59} value={daily.m}
                    onChange={(e) => setSecs(d.id, 'dailyTargetSeconds', daily.h, Math.min(59, Math.max(0, parseInt(e.target.value, 10) || 0)))}
                    className="h-9 w-20 text-center tabular" />
                  <span>{t.minutesShort}</span>
                </div>
              </div>
              <div className="rounded-lg border border-border/60 bg-muted/20 p-2.5">
                <div className="mb-2">
                  <Label>{t.maxTime} <span className="font-normal text-muted-foreground">· {t.noMax} = 0</span></Label>
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Input type="number" min={0} max={999} value={max.h}
                    onChange={(e) => setSecs(d.id, 'maxSeconds', Math.max(0, parseInt(e.target.value, 10) || 0), max.m)}
                    className="h-9 w-20 text-center tabular" />
                  <span>{t.hoursShort}</span>
                  <Input type="number" min={0} max={59} value={max.m}
                    onChange={(e) => setSecs(d.id, 'maxSeconds', max.h, Math.min(59, Math.max(0, parseInt(e.target.value, 10) || 0)))}
                    className="h-9 w-20 text-center tabular" />
                  <span>{t.minutesShort}</span>
                </div>
              </div>
            </div>
          </div>
        );
      })}
      {drafts.length === 0 && <p className="py-2 text-center text-sm text-muted-foreground">{t.noContexts}</p>}
      <div className="flex items-center gap-2.5 rounded-xl border border-dashed p-3">
        <input type="color" value={color} onChange={(e) => setColor(e.target.value)} className="size-10 shrink-0 cursor-pointer rounded-lg border border-input bg-background p-1.5" />
        <Input placeholder={t.newCtxPh} value={name} onChange={(e) => setName(e.target.value)} className="h-10 min-w-0 flex-1"
          onKeyDown={(e) => e.key === 'Enter' && name.trim() && onCreate(name.trim(), color)} />
        <Button variant="secondary" size="sm" className="h-10 shrink-0" disabled={!name.trim()} onClick={() => onCreate(name.trim(), color)}>
          <Plus /> {t.add}
        </Button>
      </div>
      <Button className="w-full" onClick={() => onUpdate(drafts)}>{t.saveChanges}</Button>
    </div>
  );
}

/* ---------------- data tab ---------------- */

function UpdateSection({ t, version, latestTag, updater, onUpdateFound }) {
  const [phase, setPhase] = React.useState(latestTag ? 'available' : 'idle'); // idle|checking|uptodate|available|downloading|downloaded
  const [info, setInfo] = React.useState(latestTag ? { latest: latestTag } : null);
  const [err, setErr] = React.useState('');

  const check = async () => {
    setPhase('checking');
    setErr('');
    try {
      const st = await updater.check(true);
      setInfo(st || null);
      if (st?.available) {
        setPhase('available');
        onUpdateFound?.(st.latest || '');
      } else {
        setPhase('uptodate');
      }
    } catch {
      setPhase('idle');
      setErr(t.checkFailed);
    }
  };

  const download = async () => {
    setPhase('downloading');
    setErr('');
    try {
      await updater.download();
      setPhase('downloaded');
    } catch (e) {
      setPhase('available');
      setErr(t.updateFailed + String((e && (e.message || e)) || e));
    }
  };

  const install = async () => {
    setErr('');
    try {
      await updater.install();
      // success quits the app — the updater script takes over from here
    } catch (e) {
      setErr(t.updateFailed + String((e && (e.message || e)) || e));
    }
  };

  return (
    <Section icon={Download} title={t.appUpdates}>
      <div className="mb-2 flex flex-wrap items-center gap-2 text-[13px]">
        <span className="text-muted-foreground">{t.appVersion}:</span>
        <span dir="ltr" className="tabular font-bold">v{version || 'dev'}</span>
        {phase === 'uptodate' && <span className="text-emerald-400">{t.upToDate}</span>}
        {phase === 'available' && info?.latest && (
          <span className="font-bold text-amber-300">{t.updateAvailable(info.latest)}</span>
        )}
      </div>
      {phase === 'downloaded' && (
        <p className="mb-2 text-[13px] text-emerald-400">{t.updateReady}</p>
      )}
      <div className="flex flex-wrap gap-2">
        {(phase === 'idle' || phase === 'uptodate') && (
          <Button variant="secondary" size="sm" onClick={check}>
            <Download /> {t.checkUpdates}
          </Button>
        )}
        {phase === 'checking' && (
          <Button variant="secondary" size="sm" disabled>
            <RotateCcw className="animate-spin" /> {t.checkingUpdates}
          </Button>
        )}
        {phase === 'available' && (
          <>
            <Button size="sm" onClick={download}>
              <Download /> {t.downloadInstall}
            </Button>
            {info?.pageUrl && (
              <Button variant="secondary" size="sm" onClick={() => { try { BrowserOpenURL(info.pageUrl); } catch { /* noop */ } }}>
                {t.viewRelease}
              </Button>
            )}
          </>
        )}
        {phase === 'downloading' && (
          <Button size="sm" disabled>
            <RotateCcw className="animate-spin" /> {t.downloading}
          </Button>
        )}
        {phase === 'downloaded' && (
          <Button size="sm" onClick={install}>
            <Power /> {t.restartToInstall}
          </Button>
        )}
      </div>
      {err && <p className="mt-2 text-xs text-red-400" dir="ltr">{err}</p>}
    </Section>
  );
}

function fmtSize(n) {
  n = n || 0;
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

function fmtDate(s) {
  if (!s) return '—';
  try {
    return new Date(s).toLocaleString();
  } catch {
    return s;
  }
}

function DataTab({ t, dbPath, onOpenFolder, drive, onImportLocal, onAskRestore, updater, version, latestTag, onUpdateFound }) {
  const [status, setStatus] = React.useState({ hasClient: false, connected: false });
  const [creds, setCreds] = React.useState({ id: '', secret: '' });
  const [backups, setBackups] = React.useState([]);
  const [busy, setBusy] = React.useState('');
  const [waiting, setWaiting] = React.useState(false);
  const [device, setDevice] = React.useState(null);
  const [showAdvanced, setShowAdvanced] = React.useState(false);
  const [err, setErr] = React.useState('');
  const [copied, setCopied] = React.useState(false);
  const pollRef = React.useRef(null);

  const refresh = React.useCallback(async () => {
    try {
      const st = await drive.getStatus();
      setStatus(st || { hasClient: false, connected: false });
      if (st?.connected) {
        try {
          setBackups((await drive.list()) || []);
        } catch {
          setBackups([]);
        }
      } else {
        setBackups([]);
      }
    } catch (e) {
      setErr(String(e));
    }
  }, [drive]);

  React.useEffect(() => {
    refresh();
    return () => { if (pollRef.current) clearInterval(pollRef.current); drive.cancelAuth?.(); };
  }, [refresh, drive]);

  const stopPoll = () => {
    if (pollRef.current) clearInterval(pollRef.current);
    pollRef.current = null;
    setWaiting(false);
    setDevice(null);
    try { drive.cancelAuth?.(); } catch { /* noop */ }
  };

  const connect = async () => {
    setErr('');
    setBusy('auth');
    try {
      const d = await drive.startAuth();
      setDevice(d);
      try { BrowserOpenURL(d.verificationUrl); } catch { /* user opens manually */ }
      setWaiting(true);
      const step = Math.max(3, d.interval || 5) * 1000;
      const deadline = Date.now() + (d.expiresIn || 900) * 1000;
      pollRef.current = setInterval(async () => {
        if (Date.now() > deadline) { stopPoll(); setBusy(''); setErr(t.codeExpired); return; }
        try {
          const r = await drive.pollAuth();
          if (!r) return;
          if (r.status === 'success') {
            stopPoll();
            setBusy('');
            await refresh();
          } else if (r.status === 'expired') {
            stopPoll();
            setBusy('');
            setErr(t.codeExpired);
          } else if (r.status === 'denied') {
            stopPoll();
            setBusy('');
            setErr(t.accessDenied);
          }
        } catch { /* keep waiting */ }
      }, step);
    } catch (e) {
      setErr(String(e));
      setBusy('');
    }
  };

  const copyPath = async () => {
    try {
      await navigator.clipboard.writeText(dbPath || '');
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      setErr(dbPath || '');
    }
  };

  return (
    <div className="space-y-3">
      <UpdateSection t={t} version={version} latestTag={latestTag} updater={updater} onUpdateFound={onUpdateFound} />
      {err && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3.5 py-2.5 text-[13px] text-red-300" dir="ltr">{err}</div>
      )}

      <Section icon={Database} title={t.dbFile}>
        <p dir="ltr" className="truncate rounded-md border bg-muted/40 px-3 py-2 text-xs tabular text-muted-foreground" title={dbPath}>
          {dbPath}
        </p>
        <div className="mt-2.5 flex flex-wrap gap-2">
          <Button variant="secondary" size="sm" onClick={copyPath}>
            {copied ? <Check /> : <Copy />} {copied ? t.copied : t.copyPath}
          </Button>
          <Button variant="secondary" size="sm" onClick={() => onOpenFolder()}>
            <FolderOpen /> {t.openFolder}
          </Button>
          <Button variant="secondary" size="sm" onClick={async () => {
            setErr('');
            const res = await onImportLocal();
            if (res && !res.ok) setErr(res.error);
          }}>
            <Download /> {t.importLocal}
          </Button>
        </div>
      </Section>

      <Section icon={Cloud} title={t.driveTitle}>
        {!status.embedded && (
          <p className="mb-2.5 text-xs leading-relaxed text-muted-foreground">{t.driveDesc}</p>
        )}
        <div className="mb-1 flex items-center gap-2">
          <span className={cn('size-2 rounded-full', status.connected ? 'bg-emerald-400' : 'bg-muted-foreground/50')} />
          <span className="text-[13px] font-bold">{status.connected ? t.connectedAs : t.notConnected}</span>
        </div>
        {(!status.embedded || showAdvanced) && (
          <>
            <div className="grid grid-cols-1 gap-2.5">
              <Field label="Client ID">
                <Input dir="ltr" value={creds.id} onChange={(e) => setCreds((c) => ({ ...c, id: e.target.value }))} placeholder="xxxx.apps.googleusercontent.com" />
              </Field>
              <Field label="Client secret">
                <Input dir="ltr" type="password" value={creds.secret} onChange={(e) => setCreds((c) => ({ ...c, secret: e.target.value }))} placeholder="••••••" />
              </Field>
            </div>
            <div className="mt-2.5 flex flex-wrap gap-2">
              <Button
                variant="secondary" size="sm" disabled={!creds.id.trim() || !creds.secret.trim() || busy === 'auth'}
                onClick={async () => {
                  setErr('');
                  try {
                    await drive.saveCreds(creds.id.trim(), creds.secret.trim());
                    await refresh();
                  } catch (e) { setErr(String(e)); }
                }}
              >
                {t.saveChanges}
              </Button>
            </div>
          </>
        )}
        <div className="mt-2.5 flex flex-wrap items-center gap-2">
          {!status.connected ? (
            <Button size="sm" disabled={busy === 'auth' || !status.hasClient} onClick={connect}>
              <Cloud /> {waiting ? t.waitingAuth : t.connectGoogle}
            </Button>
          ) : (
            <Button variant="ghost" size="sm" onClick={async () => { await drive.disconnect(); setBackups([]); await refresh(); }}>
              {t.disconnect}
            </Button>
          )}
          {waiting && device && (
            <Button variant="secondary" size="sm" onClick={() => { try { BrowserOpenURL(device.verificationUrl); } catch { /* noop */ } }}>
              {t.openGoogle}
            </Button>
          )}
          {waiting && (
            <Button variant="ghost" size="sm" onClick={() => { stopPoll(); setBusy(''); }}>{t.cancel}</Button>
          )}
          {status.embedded && (
            <button onClick={() => setShowAdvanced((v) => !v)} className="text-[11px] font-medium text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">
              {t.advancedSetup}
            </button>
          )}
        </div>
        {waiting && device && (
          <div className="mt-2.5 rounded-xl border border-primary/30 bg-primary/5 p-3.5 text-center">
            <p className="text-[11px] text-muted-foreground">{t.deviceHint}</p>
            <p dir="ltr" className="tabular mt-1 text-3xl font-black tracking-[0.2em] text-primary">{device.userCode}</p>
          </div>
        )}

        {status.connected && (
          <div className="mt-3 border-t pt-3">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-[13px] font-bold">{t.backupsTitle}</span>
              <Button size="sm" disabled={busy === 'backup'} onClick={async () => {
                setErr('');
                setBusy('backup');
                try {
                  await drive.backup();
                  await refresh();
                } catch (e) { setErr(String(e)); }
                setBusy('');
              }}>
                <Upload /> {t.backupNow}
              </Button>
            </div>
            {backups.length === 0 ? (
              <p className="py-1 text-xs text-muted-foreground">{t.noBackups}</p>
            ) : (
              <div className="max-h-44 space-y-1.5 overflow-y-auto">
                {backups.map((b) => (
                  <div key={b.id} className="flex items-center gap-2 rounded-lg border px-2.5 py-2">
                    <div className="min-w-0 flex-1">
                      <p dir="ltr" className="truncate text-xs font-semibold">{b.name}</p>
                      <p className="tabular text-[11px] text-muted-foreground">{fmtDate(b.created)} · {fmtSize(b.size)}</p>
                    </div>
                    <Button variant="secondary" size="sm" onClick={() => onAskRestore(b.id, b.name)}>{t.restore}</Button>
                    <Button variant="ghost" size="icon-sm" className="hover:bg-destructive/10 hover:text-red-400"
                      onClick={async () => { try { await drive.del(b.id); await refresh(); } catch (e) { setErr(String(e)); } }}>
                      <Trash2 size={14} />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </Section>
    </div>
  );
}

/* ---------------- MCP / AI tab ---------------- */

const MCP_TOOLS = [
  'list_tasks', 'get_task', 'create_task', 'update_task', 'move_task', 'toggle_focus',
  'delete_task', 'list_contexts', 'create_context', 'update_context', 'list_horizons', 'update_horizon',
  'create_horizon', 'delete_horizon', 'focus_list', 'stats', 'start_timer', 'stop_timer',
  'finish_task', 'active_timer', 'task_time', 'start_context_timer', 'stop_context_timer',
  'active_context_timer', 'context_time', 'get_settings', 'set_setting',
];

function aiPromptText(exe) {
  const path = exe || '';
  const entry = JSON.stringify(
    { goals: { type: 'local', command: [path, 'mcp'], enabled: true } },
    null,
    2
  );

  return `Add my Goals desktop app as a global MCP server so you can manage my tasks from ANY project.

Executable path (may contain spaces; keep it as ONE string, exactly as written):
"${path}"

Steps:
1. Open your GLOBAL MCP config (opencode: ~/.config/opencode/opencode.jsonc — create it if missing).
2. Merge this server into the top-level "mcp" object (keep every existing entry, delete nothing):
${entry}
3. Restart the session (MCP servers connect at session start), then call the "stats" tool and summarize my tasks to confirm it works.

Rules:
- The command array is [executable, "mcp"]. Do NOT split the executable path on spaces and do NOT add shell quoting inside the array.
- Never override GOALS_DB_PATH: the GUI and MCP must share the default database.
- Horizons are planning tabs (short/medium/long or a custom key) — always pass a valid one when creating tasks.
- Confirm with me before delete_task or delete_horizon.`;
}

function McpTab({ t, exePath }) {
  const [copied, setCopied] = React.useState('');
  const exe = exePath || 'goals.exe';
  const quoted = exe.includes(' ') ? `"${exe}"` : exe;
  const command = `${quoted} mcp`;
  const globalJson = `{\n  "$schema": "https://opencode.ai/config.json",\n  "mcp": {\n    "goals": {\n      "type": "local",\n      "command": ["${exe.replace(/\\/g, '\\\\')}", "mcp"],\n      "enabled": true\n    }\n  }\n}`;
  const prompt = aiPromptText(exe);

  const copy = async (key, text) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(key);
      setTimeout(() => setCopied((c) => (c === key ? '' : c)), 1500);
    } catch { /* noop */ }
  };

  const Block = ({ label, text, id }) => (
    <div>
      <div className="mb-1.5 flex items-center justify-between">
        <span className="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">{label}</span>
        <Button variant="secondary" size="sm" onClick={() => copy(id, text)}>
          {copied === id ? <Check /> : <Copy />} {copied === id ? t.copied : t.copyPath}
        </Button>
      </div>
      <pre dir="ltr" className="overflow-x-auto whitespace-pre-wrap break-all rounded-lg border bg-muted/40 px-3 py-2.5 text-left text-[11px] leading-relaxed tabular">
        {text}
      </pre>
    </div>
  );

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-bold">{t.mcpTitle}</h3>
        <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{t.mcpDesc}</p>
      </div>
      <Block id="cmd" label={t.mcpCommandLabel} text={command} />
      <Block id="global" label={t.mcpGlobalLabel} text={globalJson} />
      <div>
        <h3 className="text-sm font-bold">{t.mcpPromptTitle}</h3>
        <p className="mb-1.5 mt-1 text-xs text-muted-foreground">{t.mcpPromptDesc}</p>
        <Block id="prompt" label="PROMPT" text={prompt} />
      </div>
      <div>
        <span className="mb-1.5 block text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
          {t.mcpToolsLabel} ({MCP_TOOLS.length})
        </span>
        <div className="flex flex-wrap gap-1.5" dir="ltr">
          {MCP_TOOLS.map((name) => (
            <span key={name} className="rounded-md border bg-secondary px-2 py-0.5 text-[11px] tabular text-secondary-foreground">
              {name}
            </span>
          ))}
        </div>
      </div>
      <p className="text-[11px] text-muted-foreground">{t.mcpRestartNote}</p>
    </div>
  );
}

/* ---------------- settings window ---------------- */

const TABS = [
  { key: 'general', Icon: Globe },
  { key: 'appearance', Icon: Palette },
  { key: 'planning', Icon: CalendarRange },
  { key: 'contexts', Icon: Users },
  { key: 'prayer', Icon: Sunrise },
  { key: 'activity', Icon: Activity },
  { key: 'data', Icon: Database },
  { key: 'mcp', Icon: Bot },
];

export function SettingsSheet(props) {
  const { t, open, onClose, tab, onTab } = props;
  if (!open) return null;
  return (
    <Sheet open={open} onClose={onClose} className="max-w-2xl">
      <SheetHeader title={t.settings} onClose={onClose} />
      <div className="flex gap-1 border-b px-6 pb-0 pt-1">
        {TABS.map((tb) => (
          <button
            key={tb.key}
            onClick={() => onTab(tb.key)}
            className={cn(
              'flex items-center gap-1.5 border-b-2 px-3 pb-2.5 pt-1 text-[13px] font-bold transition-colors',
              tab === tb.key ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'
            )}
          >
            <tb.Icon size={14} />
            {t['tab' + tb.key[0].toUpperCase() + tb.key.slice(1)]}
          </button>
        ))}
      </div>
      <SheetBody>
        {tab === 'general' && (
          <GeneralTab
            t={t} lang={props.lang} onLangChange={props.onLangChange}
            autostartOn={props.autostartOn} autostartBusy={props.autostartBusy}
            autostartErr={props.autostartErr} onAutostartToggle={props.onAutostartToggle}
          />
        )}
        {tab === 'appearance' && (
          <AppearanceTab t={t} value={props.value} dirty={props.dirty} onChange={props.onChange} onReset={props.onReset} />
        )}
        {tab === 'planning' && (
          <PlanningTab t={t} lang={props.lang} horizons={props.horizons} onSave={props.onSaveHorizons} onCreate={props.onCreateHorizon} onDelete={props.onDeleteHorizon} />
        )}
        {tab === 'contexts' && (
          <ContextsTab t={t} contexts={props.contexts} onCreate={props.onCreateContext} onUpdate={props.onUpdateContexts} onDelete={props.onDeleteContext} />
        )}
        {tab === 'prayer' && (
          <PrayerTab
            t={t} lang={props.lang}
            initial={props.prayerSettings}
            onToggleEnabled={props.onTogglePrayer}
            onSave={props.onSavePrayer}
            savedFlash={props.prayerSavedFlash}
          />
        )}
        {tab === 'activity' && (
          <ActivityTab
            t={t} enabled={props.activityEnabled} busy={props.activityBusy}
            onToggleEnabled={props.onToggleActivity}
            alertsOn={props.activityAlertsOn} onToggleAlerts={props.onToggleActivityAlerts}
            dailyMin={props.activityDailyMin} sessionMin={props.activitySessionMin}
            onSaveThresholds={props.onSaveActivityThresholds}
            live={props.activityLive} onOpenInsights={props.onOpenInsights}
            onClear={props.onClearActivity} clearMsg={props.activityClearMsg}
            retentionDays={props.activityRetentionDays} onSaveRetention={props.onSaveActivityRetention}
            stats={props.activityStats} onPruneNow={props.onPruneActivityNow} pruneMsg={props.activityPruneMsg}
          />
        )}
        {tab === 'data' && (
          <DataTab
            t={t} dbPath={props.dbPath} onOpenFolder={props.onOpenFolder} drive={props.drive}
            onImportLocal={props.onImportLocal} onAskRestore={props.onAskRestore}
            updater={props.updater} version={props.version} latestTag={props.latestTag}
            onUpdateFound={props.onUpdateFound}
          />
        )}
        {tab === 'mcp' && <McpTab t={t} exePath={props.exePath} />}
      </SheetBody>
      <SheetFooter>
        <Button variant="ghost" onClick={onClose}>{t.close}</Button>
      </SheetFooter>
    </Sheet>
  );
}
