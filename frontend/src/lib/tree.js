// Tree helpers for subtasks: flat ListTasks -> nested rendering + drag ordering.
export function normalizeParentId(t) {
  const p = t?.parentId;
  if (!p || (typeof p === 'string' && !p.trim())) return null;
  return p;
}

export function sortSiblings(arr) {
  return [...arr].sort((a, b) => {
    const ao = a.sortOrder ?? 0;
    const bo = b.sortOrder ?? 0;
    if (ao !== bo) return ao - bo;
    // focused first (matches backend ORDER BY), then newest last
    const af = a.focus ? 0 : 1;
    const bf = b.focus ? 0 : 1;
    if (af !== bf) return af - bf;
    return String(a.createdAt || '') < String(b.createdAt || '') ? -1 : 1;
  });
}

// Build { roots, childrenMap } from a flat filtered list. Orphan children
// (parent filtered out) become roots so search/focus filters never lose tasks.
export function buildTree(tasks) {
  const byId = new Map();
  for (const t of tasks || []) byId.set(t.id, t);
  const childrenMap = new Map(); // parentId -> [child]
  const roots = [];
  for (const t of tasks || []) {
    const pid = normalizeParentId(t);
    if (pid && byId.has(pid) && pid !== t.id) {
      if (!childrenMap.has(pid)) childrenMap.set(pid, []);
      childrenMap.get(pid).push(t);
    } else {
      roots.push(t);
    }
  }
  for (const [k, v] of childrenMap) childrenMap.set(k, sortSiblings(v));
  return { roots: sortSiblings(roots), childrenMap, byId };
}

export function descendantIds(id, childrenMap) {
  const out = [];
  const stack = [...(childrenMap.get(id) || [])];
  const seen = new Set([id]);
  while (stack.length) {
    const cur = stack.pop();
    if (seen.has(cur.id)) continue;
    seen.add(cur.id);
    out.push(cur.id);
    for (const c of childrenMap.get(cur.id) || []) stack.push(c);
  }
  return out;
}

// Subtree time = own live elapsed + all descendants (for parent badges).
export function liveElapsedOf(task, nowMs) {
  const base = task?.elapsedSeconds || task?.elapsed || 0;
  if (task?.timerStartedAt) {
    const delta = Math.floor((nowMs - Date.parse(task.timerStartedAt)) / 1000);
    return base + Math.max(0, delta);
  }
  return base;
}

export function subtreeElapsed(task, childrenMap, nowMs) {
  let total = liveElapsedOf(task, nowMs);
  const stack = [...(childrenMap.get(task.id) || [])];
  while (stack.length) {
    const cur = stack.pop();
    total += liveElapsedOf(cur, nowMs);
    for (const c of childrenMap.get(cur.id) || []) stack.push(c);
  }
  return total;
}

export function subtreeCounts(task, childrenMap) {
  let total = 0;
  let done = 0;
  const stack = [...(childrenMap.get(task.id) || [])];
  while (stack.length) {
    const cur = stack.pop();
    total += 1;
    if (cur.status === 'done') done += 1;
    for (const c of childrenMap.get(cur.id) || []) stack.push(c);
  }
  return { total, done };
}
