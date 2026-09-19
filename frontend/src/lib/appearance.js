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
