const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];

function ensureRoleBadge() {
  const menu = $('#settings-menu');
  if (!menu) return null;
  let row = $('#session-role', menu);
  if (row) return row;
  row = document.createElement('div');
  row.id = 'session-role';
  row.className = 'session-role';
  row.innerHTML = '<span class="session-role__dot" aria-hidden="true"></span><span class="session-role__label">Checking access…</span>';
  menu.prepend(row);
  return row;
}

function applyRole(role) {
  const resolved = role === 'admin' ? 'admin' : role === 'viewer' ? 'viewer' : 'pending';
  document.documentElement.dataset.sessionRole = resolved;
  const badge = ensureRoleBadge();
  if (badge) {
    const label = $('.session-role__label', badge);
    if (label) label.textContent = resolved === 'admin' ? 'Admin access' : resolved === 'viewer' ? 'Viewer access' : 'Checking access…';
  }
  if (resolved === 'viewer') {
    const recordView = $('#record-view');
    if (recordView && !recordView.hidden) $('.nav-item[data-view="ranking"]')?.click();
  }
}

let roleRequestId = 0;
async function syncRole() {
  const shell = $('#app-shell');
  if (!shell || shell.hidden) {
    applyRole('');
    return;
  }
  const requestId = ++roleRequestId;
  try {
    const response = await fetch('/api/v1/session/role', {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    });
    if (requestId !== roleRequestId) return;
    if (!response.ok) return applyRole('viewer');
    const data = await response.json();
    applyRole(data.role);
  } catch {
    if (requestId === roleRequestId) applyRole('viewer');
  }
}

function setupRoleAwareness() {
  ensureRoleBadge();
  const shell = $('#app-shell');
  if (shell) {
    new MutationObserver(syncRole).observe(shell, {
      attributes: true,
      attributeFilter: ['hidden', 'aria-hidden'],
    });
  }
  $('#logout-btn')?.addEventListener('click', () => applyRole(''));
  syncRole();
}

function enhanceRanking() {
  const list = $('#ranking-list');
  if (!list) return;
  const rows = $$('.rank-row', list);
  rows.forEach((row) => {
    const elo = $('.rank-elo', row);
    if (!elo || row.dataset.professional === 'true') return;
    const exact = Number(elo.textContent.trim());
    if (Number.isFinite(exact)) {
      row.dataset.exactElo = exact.toFixed(2);
      const rounded = Math.round(exact);
      elo.textContent = String(rounded);
      elo.setAttribute('aria-label', `Elo ${rounded}`);
    }
    row.dataset.professional = 'true';
  });
  const status = $('#header-status');
  if (status && rows.length) status.textContent = `${rows.length} ${rows.length === 1 ? 'player' : 'players'}`;
}

function rankingEloMap() {
  const values = new Map();
  $$('#ranking-list .rank-row').forEach((row) => {
    const name = $('.rank-name', row)?.textContent.trim();
    const elo = $('.rank-elo', row)?.textContent.trim();
    if (name && elo) values.set(name.toLocaleLowerCase(), elo);
  });
  return values;
}

function setupRankingEnhancement() {
  const list = $('#ranking-list');
  if (!list) return;
  new MutationObserver(enhanceRanking).observe(list, { childList: true, subtree: true });
  enhanceRanking();
}

function enhancePlayerItem(button) {
  if (button.dataset.enhanced === 'true') return;
  const name = button.textContent.trim();
  if (!name) return;
  const elo = rankingEloMap().get(name.toLocaleLowerCase()) || '—';
  button.dataset.enhanced = 'true';
  button.dataset.playerName = name.toLocaleLowerCase();
  button.textContent = '';

  const label = document.createElement('span');
  label.className = 'sheet-item__name';
  label.textContent = name;

  const rating = document.createElement('span');
  rating.className = 'sheet-item__elo';
  rating.textContent = elo;
  rating.setAttribute('aria-label', elo === '—' ? 'Elo unavailable' : `Elo ${elo}`);

  const status = document.createElement('span');
  status.className = 'sheet-item__status';
  status.setAttribute('aria-hidden', 'true');
  button.append(label, rating, status);
}

function filterPlayers() {
  const input = $('#player-search');
  const list = $('#sheet-list');
  const empty = $('.player-search-empty');
  if (!input || !list || !empty) return;
  const query = input.value.trim().toLocaleLowerCase();
  let visible = 0;
  $$('.sheet-item', list).forEach((item) => {
    const matches = !query || (item.dataset.playerName || '').includes(query);
    item.hidden = !matches;
    if (matches) visible += 1;
  });
  empty.hidden = visible !== 0;
}

function setupPlayerSelector() {
  const sheet = $('#player-sheet');
  const list = $('#sheet-list');
  const input = $('#player-search');
  const context = $('#sheet-context');
  if (!sheet || !list || !input) return;

  const enhanceAll = () => {
    $$('.sheet-item', list).forEach(enhancePlayerItem);
    filterPlayers();
  };
  new MutationObserver(enhanceAll).observe(list, { childList: true });
  input.addEventListener('input', filterPlayers);

  document.addEventListener('click', (event) => {
    const selector = event.target.closest('.player-select');
    if (!selector) return;
    const team = selector.dataset.team || '';
    const slot = Number(selector.dataset.slot || 0) + 1;
    if (context) context.textContent = `Team ${team} · Player ${slot}`;
    input.value = '';
    requestAnimationFrame(() => {
      filterPlayers();
      input.focus({ preventScroll: true });
    });
  });
  enhanceAll();
}

function parseHistoryMeta(card) {
  const meta = $('.history-meta', card);
  if (!meta) return { label: 'Recent', time: '' };
  const parts = meta.textContent.split('·').map((part) => part.trim()).filter(Boolean);
  const rawDate = parts[0] || 'Recent';
  const time = parts.slice(1).join(' · ');
  let label = rawDate;

  if (rawDate !== 'Today') {
    const now = new Date();
    const yesterday = new Date(now);
    yesterday.setDate(now.getDate() - 1);
    const candidate = new Date(`${rawDate}, ${now.getFullYear()}`);
    if (!Number.isNaN(candidate.getTime()) && candidate.toDateString() === yesterday.toDateString()) label = 'Yesterday';
  }
  meta.textContent = time;
  meta.hidden = !time;
  return { label, time };
}

let groupingHistory = false;
function groupHistory() {
  const list = $('#history-list');
  if (!list || groupingHistory) return;
  const cards = $$('.history-card', list);
  if (!cards.length) return;
  const directCards = $$('#history-list > .history-card');
  const previousCount = Number(list.dataset.groupedCount || 0);
  if (!directCards.length && previousCount === cards.length) return;

  groupingHistory = true;
  const groups = new Map();
  cards.forEach((card) => {
    const { label } = parseHistoryMeta(card);
    if (!groups.has(label)) groups.set(label, []);
    groups.get(label).push(card);
  });

  list.replaceChildren();
  groups.forEach((groupCards, label) => {
    const section = document.createElement('section');
    section.className = 'history-group';
    const heading = document.createElement('h3');
    heading.className = 'history-group__label';
    heading.textContent = label;
    const items = document.createElement('div');
    items.className = 'history-group__items';
    groupCards.forEach((card) => items.appendChild(card));
    section.append(heading, items);
    list.appendChild(section);
  });
  list.dataset.groupedCount = String(cards.length);
  groupingHistory = false;
}

function setupHistoryGrouping() {
  const list = $('#history-list');
  if (!list) return;
  let scheduled = false;
  const schedule = () => {
    if (scheduled) return;
    scheduled = true;
    queueMicrotask(() => {
      scheduled = false;
      groupHistory();
    });
  };
  new MutationObserver(schedule).observe(list, { childList: true, subtree: true });
  schedule();
}

function recordIsDirty() {
  const hasPlayer = $$('.player-select__value').some((node) => {
    const value = node.textContent.trim().toLocaleLowerCase();
    return value && value !== 'select' && value !== 'select player';
  });
  const scoreA = Number($('#score-a')?.textContent || 0);
  const scoreB = Number($('#score-b')?.textContent || 0);
  const resultVisible = $('#result-panel') && !$('#result-panel').hidden;
  return hasPlayer || scoreA > 0 || scoreB > 0 || resultVisible;
}

function syncRecordControls() {
  const reset = $('#record-reset');
  if (reset) {
    const shouldHide = !recordIsDirty();
    if (reset.hidden !== shouldHide) reset.hidden = shouldHide;
  }
  const confirm = $('#confirm-btn');
  if (confirm && confirm.textContent.trim() === 'Confirm result') confirm.textContent = 'Confirm and record';
}

function setupRecordPolish() {
  const view = $('#record-view');
  if (!view) return;
  new MutationObserver(syncRecordControls).observe(view, {
    childList: true,
    subtree: true,
    characterData: true,
    attributes: true,
    attributeFilter: ['hidden'],
  });
  syncRecordControls();
}

function init() {
  setupRoleAwareness();
  setupRankingEnhancement();
  setupPlayerSelector();
  setupHistoryGrouping();
  setupRecordPolish();
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init, { once: true });
else init();
