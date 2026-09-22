import * as React from 'react';
import { MoonStar, AlarmClock, CheckCheck, X, BookOpenText, Expand } from 'lucide-react';
import { cn } from '../lib/utils';
import { Button } from './ui/button';
import { RADIUS } from '../lib/appearance';
import { formatClock } from '../lib/i18n';

// Full-screen prayer alert: azan sound plays from the backend (just the azan,
// never the generic chime). "Snooze N min" rings again, "I'm going to pray"
// dismisses it for this prayer, X snoozes 1 minute. Driven by App state
// (backend prayer:due event carrying an offline hadith in app lang).
export function PrayerAlert({ t, lang, event, use12h, onSnooze, onGoing, onClose }) {
  if (!event) return null;
  const name = t[event.key] || event.key;
  const use12 = use12h !== false;
  const shown = use12 ? (event.time12 || formatClock(event.time, true, lang)) : (event.time || '');
  const hadith = (event.hadith || '').trim();
  const hadithSource = (event.hadithSource || '').trim();
  return (
    <div className="fixed inset-0 z-[70] grid place-items-center overflow-y-auto bg-black/60 p-4 backdrop-blur-sm">
      <div className={cn('max-h-full w-full max-w-sm overflow-y-auto border bg-card p-5 text-center shadow-2xl', RADIUS)}>
        <div
          className="mx-auto w-fit cursor-move rounded-md px-6"
          style={{ ['--wails-draggable']: 'drag' }}
          title={t.prayerDue(name)}
        >
          <span className="mx-auto grid size-14 place-items-center rounded-2xl bg-primary/15 text-primary">
            <MoonStar size={28} />
          </span>
        </div>
        <h3 className="mt-3 text-lg font-black">{t.prayerDue(name)}</h3>
        <p className="tabular mt-1 text-sm text-muted-foreground">
          {t.prayerDueSub(shown, event.cityName || '')}
        </p>
        {hadith && (
          <div className="mt-3 rounded-xl border border-violet-500/25 bg-violet-500/5 p-3 text-start">
            <p className="mb-1 flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-violet-300">
              <BookOpenText size={13} /> {t.hadithTitle || 'Hadith'}
            </p>
            <p className="text-[13px] leading-relaxed text-foreground/90">{hadith}</p>
            {hadithSource && (
              <p className="mt-1.5 text-[11px] text-muted-foreground">— {hadithSource}</p>
            )}
          </div>
        )}
        <div className="mt-4">
          <p className="mb-1.5 flex items-center justify-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
            <AlarmClock size={13} /> {t.snooze}
          </p>
          <div className="flex justify-center gap-1.5">
            {[1, 5, 10].map((m) => (
              <button
                key={m}
                onClick={() => onSnooze(m)}
                className="tabular rounded-lg border px-3.5 py-1.5 text-xs font-bold text-muted-foreground transition-colors hover:border-primary/50 hover:text-foreground"
              >
                {t.snoozeMin(m)}
              </button>
            ))}
          </div>
        </div>
        <Button className="mt-3 w-full" onClick={onGoing}>
          <CheckCheck /> {t.goingToPray}
        </Button>
        <button
          onClick={onClose}
          className="mt-2 flex w-full items-center justify-center gap-1 rounded-md py-1 text-[11px] font-medium text-muted-foreground hover:text-foreground"
        >
          <X size={12} /> {t.dismiss}
        </button>
      </div>
    </div>
  );
}

// Compact persistent banner for the always-on-top mini window: the prayer
// alert must reach the user over whatever they are doing, even when the app
// is shrunk to its widget. Stays until "going" (or a snooze); each backend
// nag re-opens it.
export function PrayerMiniBanner({ t, lang, event, use12h, onGoing, onSnooze, onOpen }) {
  if (!event) return null;
  const name = t[event.key] || event.key;
  const use12 = use12h !== false;
  const shown = use12 ? (event.time12 || formatClock(event.time, true, lang)) : (event.time || '');
  return (
    <div className="border-b border-violet-500/30 bg-violet-500/10 px-2 py-1.5">
      <button onClick={onOpen} className="flex w-full items-center gap-1.5 text-start" title={t.prayerDue(name)}>
        <span className="grid size-6 shrink-0 place-items-center rounded-md border border-violet-500/30 bg-violet-500/10 text-violet-300">
          <MoonStar size={13} />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[12px] font-bold leading-tight">{t.prayerDue(name)}</span>
          <span className="tabular block text-[10px] leading-tight text-muted-foreground">{shown}</span>
        </span>
        <Expand size={12} className="shrink-0 text-muted-foreground" />
      </button>
      <div className="mt-1.5 flex gap-1.5">
        <button
          onClick={onGoing}
          className="flex flex-1 items-center justify-center gap-1 rounded-md bg-primary px-2 py-1 text-[11px] font-bold text-primary-foreground hover:opacity-90"
        >
          <CheckCheck size={12} /> {t.goingToPray}
        </button>
        <button
          onClick={() => onSnooze(5)}
          className="tabular rounded-md border px-2.5 py-1 text-[11px] font-bold text-muted-foreground hover:text-foreground"
        >
          {t.snoozeMin(5)}
        </button>
      </div>
    </div>
  );
}

// Compact "next prayer" line for the sidebar (only when alerts are on).
export function NextPrayerPill({ t, status, onOpen }) {
  if (!status?.enabled || !status?.nextKey) return null;
  const name = t[status.nextKey] || status.nextKey;
  return (
    <button
      onClick={onOpen}
      title={`${t.nextPrayer}: ${name}`}
      className={cn(
        'flex w-full items-center gap-2.5 rounded-lg border border-violet-500/25 bg-violet-500/5 px-2.5 py-2 text-start transition-colors hover:bg-violet-500/10',
        RADIUS
      )}
    >
      <span className="grid size-7 shrink-0 place-items-center rounded-md border border-violet-500/30 bg-violet-500/10 text-violet-300">
        <MoonStar size={14} />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-[13px] font-semibold">{name}</span>
        <span className="tabular block text-[11px] leading-none text-muted-foreground">
          {status.inSeconds != null ? t.prayerIn(formatCountdown(status.inSeconds)) : ''}
        </span>
      </span>
    </button>
  );
}

function formatCountdown(s) {
  s = Math.max(0, Math.floor(s || 0));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const mm = String(m).padStart(2, '0');
  const ss = String(sec).padStart(2, '0');
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
}
