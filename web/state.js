export const state = {
  authenticated: false,
  role: 'viewer',
  checkingSession: true,
  rankings: [],
  matches: [],
  nextMatch: null,
  players: [],
  selected: { A0: null, A1: null, B0: null, B1: null },
  scores: { a: 0, b: 0 },
  activeView: 'ranking',
  loading: { ranking: false, history: false, match: false, login: false, rollback: false, nextMatch: false },
  error: { ranking: null, history: null, match: null, login: null },
  lastResult: null,
  lastUpdated: null,
};

const listeners = new Set();
export function subscribe(fn) { listeners.add(fn); return () => listeners.delete(fn); }
export function notify() { listeners.forEach(fn => fn(state)); }
export function patch(updater) {
  if (typeof updater === 'function') updater(state);
  else Object.assign(state, updater);
  notify();
}
export function resetSelection() {
  state.selected = { A0: null, A1: null, B0: null, B1: null };
  state.scores = { a: 0, b: 0 };
  notify();
}
export function getSelectedPlayers() {
  return [state.selected.A0, state.selected.A1, state.selected.B0, state.selected.B1];
}
export function isSelectionComplete() {
  const vals = getSelectedPlayers();
  return vals.every(Boolean) && new Set(vals.map(v => v.toLowerCase())).size === 4;
}
export function getDisabledPlayers() {
  const set = new Set(getSelectedPlayers().filter(Boolean).map(v => v.toLowerCase()));
  return set;
}
