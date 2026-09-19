// Appearance engine — waffiy-style visuals for Goals.
// Settings persist in SQLite (backend) + localStorage (instant, flicker-free).

export const DEFAULT_APPEARANCE = {
  theme: 'dark', // dark | light | system
  accent: 'indigo', // indigo | sky | teal | emerald | amber | orange | rose | pink | violet
  font: 'Cairo', // Cairo | Tajawal | system
  fontSize: 'md', // sm | md | lg | xl
  radius: 'balanced', // none | sharp | balanced | soft | round
  cardStyle: 'elevated', // elevated | bordered | flat | glass
  shadows: 'on', // on | off
  density: 'comfortable', // comfortable | compact | spacious
  animations: 'on', // on | off
  glass: 'off', // off | on — frosted sidebar
  cardDesc: 'on', // off | on — description preview on cards
};

export const ACCENTS = {
  indigo: '239 84% 67%',
  sky: '199 89% 48%',
  teal: '174 72% 40%',
  emerald: '160 84% 39%',
  amber: '38 92% 50%',
  orange: '24 95% 53%',
  rose: '350 89% 60%',
  pink: '330 81% 60%',
  violet: '271 81% 56%',
};

export const RADII = { none: '0', sharp: '0.3rem', balanced: '0.65rem', soft: '1rem', round: '1.4rem' };
export const FONT_SIZES = { sm: '14px', md: '16px', lg: '18px', xl: '20px' };

export const FONTS = {
  Cairo: "'Cairo', ui-sans-serif, system-ui, sans-serif",
  Tajawal: "'Tajawal', ui-sans-serif, system-ui, sans-serif",
  system: "ui-sans-serif, system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif",
};

const LS_KEY = 'goals-appearance';

export function loadLocalAppearance() {
  try {
    const raw = localStorage.getItem(LS_KEY);
    if (raw) return { ...DEFAULT_APPEARANCE, ...JSON.parse(raw) };
  } catch {}
  return { ...DEFAULT_APPEARANCE };
}

export function saveLocalAppearance(a) {
  try {
    localStorage.setItem(LS_KEY, JSON.stringify(a));
  } catch {}
}

let systemListener = null;

export function applyAppearance(a) {
  const s = { ...DEFAULT_APPEARANCE, ...a };
  const root = document.documentElement;

  // theme mode
  const mq = window.matchMedia?.('(prefers-color-scheme: dark)');
  const wantDark = s.theme === 'dark' || (s.theme === 'system' && !!mq?.matches);
  root.classList.toggle('dark', wantDark);
  if (systemListener && mq?.removeEventListener) mq.removeEventListener('change', systemListener);
  systemListener = null;
  if (s.theme === 'system' && mq?.addEventListener) {
    systemListener = () => root.classList.toggle('dark', mq.matches);
    mq.addEventListener('change', systemListener);
  }

  // accent + radius + font size + family
  root.style.setProperty('--primary', ACCENTS[s.accent] || ACCENTS.indigo);
  root.style.setProperty('--ring', ACCENTS[s.accent] || ACCENTS.indigo);
  root.style.setProperty('--radius', RADII[s.radius] || RADII.balanced);
  root.style.fontSize = FONT_SIZES[s.fontSize] || FONT_SIZES.md;
  document.body.style.fontFamily = FONTS[s.font] || FONTS.Cairo;

  // dataset flags consumed by CSS
  root.dataset.card = s.cardStyle;
  root.dataset.shadows = s.shadows;
  root.dataset.density = s.density;
  root.dataset.anim = s.animations;
  root.dataset.glass = s.glass;
  root.dataset.carddesc = s.cardDesc;
}

// Backend <-> appearance key mapping (settings table stores flat keys).
const PREFIX = 'appearance.';
export function fromSettingsMap(map) {
  const out = { ...DEFAULT_APPEARANCE };
  for (const k of Object.keys(DEFAULT_APPEARANCE)) {
    if (map && map[PREFIX + k]) out[k] = map[PREFIX + k];
  }
  return out;
}
export function settingsDiff(prev, next) {
  const changed = [];
  for (const k of Object.keys(DEFAULT_APPEARANCE)) {
    if ((prev[k] || '') !== (next[k] || '')) changed.push([PREFIX + k, next[k]]);
  }
  return changed;
}

// ---------- class helpers: single source for kanban / table / filters / badges ----------
// Every board surface consumes these, so Settings → Appearance visibly affects
// task cards, columns, the table, status-filter chips and all badges.
// `a` accepts a full or partial appearance object (defaults fill the gaps).

/** card/column/table corner radius — follows the radius setting */
export const RADIUS = 'rounded-[var(--radius)]';
/** half radius for small inner badges */
export const RADIUS_SM = 'rounded-[calc(var(--radius)/1.5)]';

const DENSITY_CLASSES = {
  compact: {
    card: 'p-2.5',
    colBody: 'gap-1.5 p-1.5',
    cell: 'px-3 py-1.5',
    chip: 'px-2.5 py-0.5 text-[11px]',
    badge: 'px-1.5 py-px text-[10px]',
  },
  comfortable: {
    card: 'p-4',
    colBody: 'gap-2.5 p-2',
    cell: 'px-4 py-2.5',
    chip: 'px-3 py-1 text-xs',
    badge: 'px-2 py-0.5 text-[11px]',
  },
  spacious: {
    card: 'p-5',
    colBody: 'gap-3.5 p-3',
    cell: 'px-5 py-3.5',
    chip: 'px-4 py-1.5 text-[13px]',
    badge: 'px-2.5 py-1 text-xs',
  },
};

export function normalizeAppearance(a) {
  return { ...DEFAULT_APPEARANCE, ...(a || {}) };
}

/** density class set for the given appearance */
export function density(a) {
  const d = normalizeAppearance(a).density;
  return DENSITY_CLASSES[d] || DENSITY_CLASSES.comfortable;
}

/** surface treatment for cards / columns / tables (cardStyle + shadows switch) */
export function surfClass(a) {
  const s = normalizeAppearance(a);
  const shadow = s.shadows === 'off' ? 'shadow-none' : 'shadow-sm';
  // NOTE: keep the `surf` marker — index.css layers cardStyle overrides on top.
  if (s.cardStyle === 'flat') return 'surf border-transparent bg-transparent shadow-none';
  if (s.cardStyle === 'bordered') return 'surf border bg-card shadow-none';
  if (s.cardStyle === 'glass') return `surf border bg-card/55 backdrop-blur-md ${shadow}`;
  return `surf border bg-card ${shadow}`; // elevated
}

/** hover lift + transition, honoring the animations switch */
export function motionClass(a) {
  if (normalizeAppearance(a).animations === 'off') return '';
  return 'transition-all hover:-translate-y-px';
}

/** plain color transition, honoring the animations switch */
export function animClass(a) {
  return normalizeAppearance(a).animations === 'off' ? '' : 'transition-colors';
}

/** hover elevation, honoring the shadows switch */
export function hoverShadowClass(a) {
  return normalizeAppearance(a).shadows === 'off' ? '' : 'hover:shadow-lg hover:shadow-primary/5';
}

/** status-filter chip — density-aware padding/text, accent-aware active state */
export function filterChipClass(a, isActive) {
  const d = density(a);
  const base = `${RADIUS} border font-medium ${d.chip} ${animClass(a)}`;
  return isActive
    ? `${base} border-primary/50 bg-primary/15 text-primary`
    : `${base} border-border bg-transparent text-muted-foreground hover:border-muted-foreground/40 hover:text-foreground`;
}
