import * as React from 'react';
import {
  Moon, Sun, Monitor, Palette, Type, Square, Layers, Box,
  Sparkles, Maximize2, RotateCcw, Plus, Trash2, CalendarRange, Users,
} from 'lucide-react';
import { cn } from '../lib/utils';
import { Button } from './ui/button';
import { Sheet, SheetHeader, SheetBody, SheetFooter } from './ui/sheet';
import { Input, Label } from './ui/form';
import { ACCENTS, FONTS } from '../lib/appearance';
import { horizonName } from '../lib/i18n';

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
        <div className="surf tcard rounded-xl border bg-card p-3">
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
              className={cn('rounded-full border px-3.5 py-1.5 text-[13px] transition-all',
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
            <span className="tabular rounded-full bg-primary/10 px-2 py-0.5 text-[11px] font-semibold text-primary">~{r.defaultDays}</span>
            {!BUILTINS.has(r.key) && (
              <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-bold text-muted-foreground">{t.customTab}</span>
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

  return (
    <div className="space-y-2.5">
      {drafts.map((d) => (
        <div key={d.id} className="flex items-center gap-2.5">
          <input type="color" value={d.color} onChange={(e) => set(d.id, 'color', e.target.value)}
            className="size-9 shrink-0 cursor-pointer rounded-md border border-input bg-background p-1" />
          <Input value={d.name} onChange={(e) => set(d.id, 'name', e.target.value)} className="flex-1" />
          <Button variant="ghost" size="icon-sm" className="shrink-0 hover:bg-destructive/10 hover:text-red-400" onClick={() => onDelete(d)}>
            <Trash2 size={14} />
          </Button>
        </div>
      ))}
      {drafts.length === 0 && <p className="py-2 text-center text-sm text-muted-foreground">{t.noContexts}</p>}
      <div className="flex items-center gap-2.5 rounded-lg border border-dashed p-2.5">
        <input type="color" value={color} onChange={(e) => setColor(e.target.value)} className="size-9 shrink-0 cursor-pointer rounded-md border border-input bg-background p-1" />
        <Input placeholder={t.newCtxPh} value={name} onChange={(e) => setName(e.target.value)} className="flex-1"
          onKeyDown={(e) => e.key === 'Enter' && name.trim() && onCreate(name.trim(), color)} />
        <Button variant="secondary" size="sm" disabled={!name.trim()} onClick={() => onCreate(name.trim(), color)}>
          <Plus /> {t.add}
        </Button>
      </div>
      <Button className="w-full" onClick={() => onUpdate(drafts)}>{t.saveChanges}</Button>
    </div>
  );
}

/* ---------------- settings window ---------------- */

const TABS = [
  { key: 'appearance', Icon: Palette },
  { key: 'planning', Icon: CalendarRange },
  { key: 'contexts', Icon: Users },
];

export function SettingsSheet(props) {
  const { t, open, onClose, tab, onTab } = props;
  if (!open) return null;
  return (
    <Sheet open={open} onClose={onClose}>
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
        {tab === 'appearance' && (
          <AppearanceTab t={t} value={props.value} dirty={props.dirty} onChange={props.onChange} onReset={props.onReset} />
        )}
        {tab === 'planning' && (
          <PlanningTab t={t} lang={props.lang} horizons={props.horizons} onSave={props.onSaveHorizons} onCreate={props.onCreateHorizon} onDelete={props.onDeleteHorizon} />
        )}
        {tab === 'contexts' && (
          <ContextsTab t={t} contexts={props.contexts} onCreate={props.onCreateContext} onUpdate={props.onUpdateContexts} onDelete={props.onDeleteContext} />
        )}
      </SheetBody>
      <SheetFooter>
        <Button variant="ghost" onClick={onClose}>{t.close}</Button>
      </SheetFooter>
    </Sheet>
  );
}
