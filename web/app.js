import { API } from './api.js';
import { state, patch, resetSelection, getSelectedPlayers, isSelectionComplete, getDisabledPlayers } from './state.js';

const $ = (s, r=document) => r.querySelector(s);
const $$ = (s, r=document) => [...r.querySelectorAll(s)];

const els = {
  loadingScreen: $('#loading-screen'),
  loginScreen: $('#login-screen'),
  appShell: $('#app-shell'),
  loginForm: $('#login-form'),
  password: $('#password'),
  togglePw: $('#toggle-password'),
  loginBtn: $('#login-btn'),
  loginError: $('#login-error'),
  loginCancel: $('#login-cancel'),
  settingsBtn: $('#settings-btn'),
  settingsMenu: $('#settings-menu'),
  refreshData: $('#refresh-data'),
  manageNextMatch: $('#manage-next-match'),
  logoutBtn: $('#logout-btn'),
  rankingView: $('#ranking-view'),
  recordView: $('#record-view'),
  historyView: $('#history-view'),
  rankingList: $('#ranking-list'),
  rankingSkeleton: $('#ranking-skeleton'),
  rankingEmpty: $('#ranking-empty'),
  rankingError: $('#ranking-error'),
  rankingUpdated: $('#ranking-updated'),
  rankingRefresh: $('#ranking-refresh'),
  nextMatchCard: $('#next-match-card'),
  nextMatchDate: $('#next-match-date'),
  nextMatchTime: $('#next-match-time'),
  nextMatchLocation: $('#next-match-location'),
  nextMatchSheet: $('#next-match-sheet'),
  nextMatchForm: $('#next-match-form'),
  nextMatchInputDate: $('#next-match-input-date'),
  nextMatchInputTime: $('#next-match-input-time'),
  nextMatchInputLocation: $('#next-match-input-location'),
  nextMatchFormError: $('#next-match-form-error'),
  nextMatchSave: $('#next-match-save'),
  nextMatchClear: $('#next-match-clear'),
  historyList: $('#history-list'),
  historySkeleton: $('#history-skeleton'),
  historyEmpty: $('#history-empty'),
  historyError: $('#history-error'),
  historyRefresh: $('#history-refresh'),
  scoreA: $('#score-a'),
  scoreB: $('#score-b'),
  matchSets: $('#match-sets'),
  winnerPreview: $('#winner-preview'),
  recordBtn: $('#record-btn'),
  recordError: $('#record-error'),
  recordReset: $('#record-reset'),
  managePlayersBtn: $('#manage-players-btn'),
  playersSheet: $('#players-sheet'),
  addPlayerForm: $('#add-player-form'),
  addPlayerName: $('#add-player-name'),
  addPlayerElo: $('#add-player-elo'),
  playerManagementList: $('#player-management-list'),
  profileSheet: $('#profile-sheet'),
  profileContent: $('#profile-content'),
  resultPanel: $('#result-panel'),
  resultSummary: $('#result-summary'),
  resultChanges: $('#result-changes'),
  viewRankingBtn: $('#view-ranking-btn'),
  recordAnotherBtn: $('#record-another-btn'),
  shareResultBtn: $('#share-result-btn'),
  playerSheet: $('#player-sheet'),
  sheetList: $('#sheet-list'),
  confirmSheet: $('#confirm-sheet'),
  confirmContent: $('#confirm-content'),
  confirmBtn: $('#confirm-btn'),
  rollbackSheet: $('#rollback-sheet'),
  rollbackContent: $('#rollback-content'),
  rollbackConfirmBtn: $('#rollback-confirm-btn'),
  toastRoot: $('#toast-root'),
};

let sheetTrigger = null;
let activeSheet = null;
let submitting = false;
let currentSheetTarget = null; // {team, slot}
let pendingRollbackId = null;

// Helpers
function toast(msg, type='success', timeout=2800) {
  const t = document.createElement('div');
  t.className = `toast toast--${type}`;
  t.textContent = msg;
  els.toastRoot.appendChild(t);
  setTimeout(() => { t.classList.add('is-leaving'); setTimeout(()=>t.remove(), 180); }, timeout);
}
function showLoginError(msg) {
  els.loginError.textContent = msg;
  els.loginError.hidden = !msg;
}
function showRecordError(msg) {
  els.recordError.textContent = msg;
  els.recordError.hidden = !msg;
}
function setLoginLoading(v) {
  state.loading.login = v;
  els.loginBtn.disabled = v;
  els.loginBtn.textContent = v ? 'Unlocking…' : 'Unlock recording';
}
function openSheet(sheetEl, trigger) {
  sheetTrigger = trigger || document.activeElement;
  activeSheet = sheetEl;
  sheetEl.hidden = false;
  sheetEl.setAttribute('aria-hidden','false');
  // focus trap: focus first button
  const focusable = sheetEl.querySelector('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
  if (focusable) focusable.focus();
  document.addEventListener('keydown', onSheetKey);
}
function closeSheet(sheetEl) {
  sheetEl.hidden = true;
  sheetEl.setAttribute('aria-hidden','true');
  if (activeSheet === sheetEl) activeSheet = null;
  document.removeEventListener('keydown', onSheetKey);
  if (sheetTrigger) { try{ sheetTrigger.focus(); }catch{} sheetTrigger=null; }
}
function onSheetKey(e) {
  if (e.key === 'Escape' && activeSheet) {
    e.preventDefault();
    closeSheet(activeSheet);
  }
  if (e.key === 'Tab' && activeSheet) {
    const focusable = [...activeSheet.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])')].filter(el=>!el.disabled && el.offsetParent!==null);
    if (!focusable.length) return;
    const first = focusable[0], last = focusable[focusable.length-1];
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
  }
}
function escapeHtml(s) { return String(s).replace(/[&<>"']/g, c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }

function parseSetScores(value) {
  if (!value.trim()) return null;
  const sets = value.split(',').map(part => part.trim()).filter(Boolean).map((part) => {
    const match = /^(\d{1,2})\s*[-–:]\s*(\d{1,2})$/.exec(part);
    if (!match || match[1] === match[2]) throw new Error('Use match scores like 6-2, 2-6; ties are not allowed');
    return { team1: Number(match[1]), team2: Number(match[2]) };
  });
  if (!sets.length) throw new Error('Enter at least one set');
  return sets;
}
function getMatchScores() {
  const sets = parseSetScores(els.matchSets.value);
  if (!sets) return { score1: state.scores.a, score2: state.scores.b, sets: null };
  return { score1: sets[0].team1, score2: sets[0].team2, sets };
}

// Auth
async function checkSession() {
  try {
    const data = await API.getSession();
    state.authenticated = !!data.authenticated;
    if (state.authenticated) {
      const access = await API.getSessionRole();
      state.role = access.role === 'admin' ? 'admin' : 'viewer';
    }
  } catch (e) {
    state.authenticated = false;
    state.role = 'viewer';
  } finally {
    state.checkingSession = false;
    renderAuth();
    await loadInitialData();
  }
}
function renderAuth() {
  document.documentElement.dataset.sessionRole = state.role;
  els.loginScreen.hidden = true;
  els.loginScreen.setAttribute('aria-hidden','true');
  els.appShell.hidden = false;
  els.appShell.setAttribute('aria-hidden','false');
}
function showRecordingLogin() {
  els.loginScreen.hidden = false;
  els.loginScreen.setAttribute('aria-hidden','false');
  els.password.value = '';
  showLoginError('');
  requestAnimationFrame(() => els.password.focus());
}
function closeRecordingLogin() {
  els.loginScreen.hidden = true;
  els.loginScreen.setAttribute('aria-hidden','true');
  els.password.value = '';
  showLoginError('');
}
function handleSessionExpired() {
  state.authenticated = false;
  state.role = 'viewer';
  state.players = [];
  state.lastResult = null;
  renderAuth();
  switchView('ranking');
  toast('Recording access expired.', 'error', 3500);
  closeSheet(els.confirmSheet);
  closeSheet(els.rollbackSheet);
  closeSheet(els.playerSheet);
  closeSheet(els.nextMatchSheet);
  els.settingsMenu.hidden = true;
  els.settingsBtn.setAttribute('aria-expanded','false');
}

function formatNextMatchDate(value) {
  const date = new Date(`${value}T00:00:00`);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' });
}

function formatNextMatchTime(value) {
  const date = new Date(`1970-01-01T${value}:00`);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
}

function renderNextMatch() {
  const next = state.nextMatch;
  els.nextMatchCard.hidden = !next;
  els.nextMatchClear.hidden = !next;
  if (!next) return;
  els.nextMatchDate.textContent = formatNextMatchDate(next.date);
  els.nextMatchTime.textContent = formatNextMatchTime(next.time);
  els.nextMatchLocation.textContent = next.location;
}

async function loadNextMatch() {
  if (state.loading.nextMatch) return;
  state.loading.nextMatch = true;
  try {
    const data = await API.getNextMatch();
    state.nextMatch = data.next_match || null;
  } catch (error) {
    state.nextMatch = null;
    console.warn('Could not load next match', error);
  } finally {
    state.loading.nextMatch = false;
    renderNextMatch();
  }
}

function openNextMatchEditor(trigger) {
  if (state.role !== 'admin') return;
  const next = state.nextMatch;
  els.nextMatchInputDate.value = next?.date || '';
  els.nextMatchInputTime.value = next?.time || '';
  els.nextMatchInputLocation.value = next?.location || '';
  els.nextMatchFormError.hidden = true;
  els.nextMatchFormError.textContent = '';
  els.nextMatchClear.hidden = !next;
  openSheet(els.nextMatchSheet, trigger);
}

// Navigation
function switchView(name) {
  if (name === 'record' && state.role !== 'admin') {
    showRecordingLogin();
    return;
  }
  state.activeView = name;
  els.rankingView.hidden = name !== 'ranking';
  els.recordView.hidden = name !== 'record';
  els.historyView.hidden = name !== 'history';
  $$('.nav-item').forEach(b => {
    const is = b.dataset.view === name;
    b.classList.toggle('is-active', is);
    if (is) b.setAttribute('aria-current','page'); else b.removeAttribute('aria-current');
  });
  if (name === 'ranking') loadRankings();
  if (name === 'history') loadHistory();
}

// Ranking
async function loadRankings() {
  if (state.loading.ranking) return;
  state.loading.ranking = true;
  state.error.ranking = null;
  els.rankingSkeleton.hidden = false;
  els.rankingError.hidden = true;
  els.rankingEmpty.hidden = true;
  const btn = els.rankingRefresh;
  if (btn) { btn.classList.add('is-loading'); btn.disabled = true; }
  try {
    const data = await API.getRankings();
    state.rankings = data.rankings || [];
    state.players = state.rankings.map(r => r.name);
    state.lastUpdated = new Date();
    renderRanking();
    if (btn) { btn.classList.remove('is-loading'); btn.classList.add('is-success'); setTimeout(()=>btn.classList.remove('is-success'), 1500); }
  } catch (e) {
    if (e.status === 401) return handleSessionExpired();
    state.error.ranking = e.message;
    els.rankingError.hidden = false;
    toast(e.message, 'error');
    if (btn) btn.classList.remove('is-loading');
  } finally {
    state.loading.ranking = false;
    els.rankingSkeleton.hidden = true;
    if (btn) btn.disabled = false;
  }
}
function renderRanking() {
  els.rankingList.innerHTML = '';
  if (!state.rankings.length) {
    els.rankingEmpty.hidden = false;
    els.rankingUpdated.textContent = '';
    return;
  }
  els.rankingEmpty.hidden = true;
  els.rankingUpdated.textContent = state.lastUpdated ? `Updated ${state.lastUpdated.toLocaleTimeString([], {hour:'numeric',minute:'2-digit'})}` : '';
  state.rankings.forEach((p) => {
    const row = document.createElement('div');
    row.className = 'rank-row';
    row.setAttribute('role','listitem');
    if (p.position === 1) row.classList.add('rank-row--gold');
    else if (p.position === 2) row.classList.add('rank-row--silver');
    else if (p.position === 3) row.classList.add('rank-row--bronze');
    const initial = (p.name.trim()[0] || '?').toUpperCase();
    const elo = Number(p.elo).toFixed(2);
    row.innerHTML = `
      <span class="rank-pos" aria-label="Position ${p.position}">${p.position}</span>
      <span class="rank-avatar" aria-hidden="true">${escapeHtml(initial)}</span>
      <span class="rank-main"><span class="rank-name">${escapeHtml(p.name)}</span>${p.rank_change ? `<span class="rank-movement rank-movement--${p.rank_change > 0 ? 'up' : 'down'}" aria-label="Moved ${p.rank_change > 0 ? 'up' : 'down'} ${Math.abs(p.rank_change)} ${Math.abs(p.rank_change) === 1 ? 'place' : 'places'}">${p.rank_change > 0 ? '↑' : '↓'}${Math.abs(p.rank_change)}</span>` : ''}</span>
      <span class="rank-elo" aria-label="Elo ${elo}">${elo}</span>
    `;
    row.dataset.playerName = p.name.trim().toLocaleLowerCase();
    row.classList.add('rank-row--profile');
    row.addEventListener('click', () => openPlayerProfile(p.name, row));
    els.rankingList.appendChild(row);
  });
}

async function openPlayerProfile(name, trigger) {
  try {
    const profile = await API.getPlayerProfile(name);
    const analytics = profile.analytics;
    const lastPlayed = profile.last_played ? new Date(profile.last_played).toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' }) : 'Not yet played';
    const streak = analytics.streak ? `${analytics.streak.count} ${analytics.streak.kind} streak` : 'No streak yet';
    const form = analytics.recent_form.length ? analytics.recent_form.map(result => `<i class="profile-form--${result === 'W' ? 'win' : 'loss'}">${result}</i>`).join('') : '—';
    els.profileContent.innerHTML = `<div class="profile-hero"><strong>${escapeHtml(profile.name)}</strong><span>${Number(profile.elo).toFixed(0)} Elo · #${profile.position} · Last played ${lastPlayed}</span></div><div class="profile-stats"><span><b>${profile.wins}</b> wins</span><span><b>${profile.losses}</b> losses</span><span><b>${analytics.win_rate}%</b> win rate</span></div><div class="profile-insights"><span><b class="profile-form">${form}</b><small>Last 5 form</small></span><span><b>${analytics.matches_this_month}</b><small>Matches this month</small></span><span><b>${Number(analytics.best_elo).toFixed(0)} Elo</b><small>Best Elo</small></span><span><b>${streak}</b><small>Current form</small></span><span><b>${analytics.average_delta >= 0 ? '+' : ''}${analytics.average_delta} Elo</b><small>Average per match</small></span><span><b>${analytics.best_teammate ? escapeHtml(analytics.best_teammate.name) : '—'}</b><small>Best teammate</small></span></div><p class="profile-heading">Recent matches</p><div class="profile-matches">${profile.matches.length ? profile.matches.slice(0, 10).map(match => `<div>${escapeHtml(match.winners.join(' & '))} <b>${match.winner_score}–${match.loser_score}</b> ${escapeHtml(match.losers.join(' & '))}</div>`).join('') : 'No matches yet.'}</div>`;
    openSheet(els.profileSheet, trigger);
  } catch (error) { toast(error.message || 'Could not load profile', 'error'); }
}


function renderPlayerManagement() {
  els.playerManagementList.replaceChildren();
  if (!state.rankings.length) {
    els.playerManagementList.textContent = 'No players yet.';
    return;
  }
  state.rankings.forEach((player) => {
    const item = document.createElement('div');
    item.className = 'player-manager__item';
    const name = document.createElement('span');
    name.className = 'player-manager__name';
    name.textContent = player.name;
    const elo = document.createElement('span');
    elo.className = 'player-manager__rating';
    elo.textContent = `${Number(player.elo).toFixed(0)} Elo`;
    const remove = document.createElement('button');
    remove.className = 'player-manager__remove';
    remove.type = 'button';
    remove.textContent = 'Remove';
    remove.setAttribute('aria-label', `Remove ${player.name}`);
    remove.addEventListener('click', async () => {
      if (!window.confirm(`Remove ${player.name}? This cannot be undone.`)) return;
      remove.disabled = true;
      try {
        await API.deletePlayer(player.name);
        Object.entries(state.selected).forEach(([key, selected]) => {
          if (selected && selected.toLocaleLowerCase() === player.name.toLocaleLowerCase()) state.selected[key] = null;
        });
        await loadRankings();
        renderSelectors();
        renderPlayerManagement();
        toast(`${player.name} removed`, 'success');
      } catch (error) {
        if (error.status === 401) return handleSessionExpired();
        toast(error.message || 'Could not remove player', 'error');
      } finally {
        remove.disabled = false;
      }
    });
    const edit = document.createElement('button');
    edit.className = 'player-manager__remove player-manager__edit';
    edit.type = 'button';
    edit.textContent = 'Edit';
    edit.addEventListener('click', async () => {
      const nextName = window.prompt('Player name', player.name);
      if (nextName === null) return;
      const change = window.prompt('Elo adjustment (for example +25 or -10)', '0');
      try {
        if (nextName.trim() && nextName.trim() !== player.name) await API.renamePlayer(player.name, nextName.trim());
        const delta = Number(change || 0);
        if (!Number.isFinite(delta)) throw new Error('Enter a valid Elo adjustment');
        if (delta) await API.adjustPlayer(nextName.trim() || player.name, delta);
        await loadRankings(); renderSelectors(); renderPlayerManagement(); toast('Player updated', 'success');
      } catch (error) { toast(error.message || 'Could not update player', 'error'); }
    });
    item.append(name, elo, edit, remove);
    els.playerManagementList.appendChild(item);
  });
}

async function openPlayerManagement(trigger) {
  if (document.documentElement.dataset.sessionRole !== 'admin') return;
  await loadRankings();
  renderPlayerManagement();
  openSheet(els.playersSheet, trigger);
}

// Players & Record
async function ensurePlayers() {
  if (state.players.length) return;
  try {
    const data = await API.getPlayers().catch(()=>null);
    if (data && data.players) {
      state.players = data.players.map(p => p.name);
    } else {
      const r = await API.getRankings();
      state.players = (r.rankings||[]).map(p=>p.name);
    }
  } catch (e) {
    if (e.status===401) handleSessionExpired();
  }
}
function renderSelectors() {
  $$('.player-select').forEach(btn => {
    const team = btn.dataset.team;
    const slot = btn.dataset.slot;
    const key = team + slot;
    const val = state.selected[key];
    const valueEl = btn.querySelector('.player-select__value');
    valueEl.textContent = val || 'Select';
    btn.setAttribute('aria-label', `Select Player ${Number(slot)+1} for Team ${team}${val ? ': '+val : ''}`);
  });
  updateRecordButton();
  updateWinnerPreview();
}
function updateWinnerPreview() {
  let a = state.scores.a, b = state.scores.b;
  let matches = null;
  try {
    const scores = getMatchScores();
    ({ score1: a, score2: b } = scores);
    matches = scores.sets;
  } catch { /* show the live score until the scores are valid */ }
  const aCtrl = document.querySelector('[data-score="A"]');
  const bCtrl = document.querySelector('[data-score="B"]');
  els.winnerPreview.classList.remove('winner-preview--accent', 'winner-preview--muted');
  if (matches) {
    aCtrl.classList.remove('is-winner');
    bCtrl.classList.remove('is-winner');
    els.winnerPreview.textContent = `${matches.length} ${matches.length === 1 ? 'match' : 'matches'} ready`;
    els.winnerPreview.classList.add('winner-preview--accent');
    return;
  }
  aCtrl.classList.toggle('is-winner', a > b);
  bCtrl.classList.toggle('is-winner', b > a);
  if (a === b) {
    els.winnerPreview.textContent = a===0 && b===0 ? '' : 'Tied — adjust score to record';
    els.winnerPreview.classList.add('winner-preview--muted');
  } else {
    const winner = a > b ? 'Team A wins' : 'Team B wins';
    els.winnerPreview.textContent = winner;
    els.winnerPreview.classList.add('winner-preview--accent');
  }
}
function updateRecordButton() {
  const complete = isSelectionComplete();
  let tie = state.scores.a === state.scores.b;
  try { const scores = getMatchScores(); tie = scores.score1 === scores.score2; } catch { tie = true; }
  const disabled = !complete || tie || submitting;
  els.recordBtn.disabled = disabled;
  if (!complete && !state.lastResult) {
    showRecordError('');
  } else if (tie && complete) {
    showRecordError('Tied scores cannot be submitted');
  } else {
    showRecordError('');
  }
}
function openPlayerSheet(team, slot, trigger) {
  currentSheetTarget = { team, slot };
  ensurePlayers().then(()=> {
    els.sheetList.innerHTML = '';
    const disabled = getDisabledPlayers();
    const currentKey = team + slot;
    const currentVal = state.selected[currentKey];
    // Filter out other selected players
    state.players.forEach(name => {
      const lower = name.toLowerCase();
      const isDisabled = disabled.has(lower) && lower !== (currentVal||'').toLowerCase();
      const btn = document.createElement('button');
      btn.className = 'sheet-item';
      btn.textContent = name;
      btn.disabled = isDisabled;
      btn.setAttribute('role','option');
      if (name === currentVal) btn.classList.add('is-selected');
      btn.addEventListener('click', () => {
        state.selected[currentKey] = name;
        closeSheet(els.playerSheet);
        renderSelectors();
      });
      els.sheetList.appendChild(btn);
    });
    if (!state.players.length) {
      els.sheetList.innerHTML = '<p class="player-search-empty">No players available</p>';
    }
    openSheet(els.playerSheet, trigger);
  });
}

// Score
function changeScore(team, delta) {
  const key = team === 'A' ? 'a' : 'b';
  const next = Math.max(0, state.scores[key] + delta);
  state.scores[key] = next;
  if (team === 'A') els.scoreA.textContent = String(next);
  else els.scoreB.textContent = String(next);
  updateWinnerPreview();
  updateRecordButton();
}

// Record flow
function showConfirm() {
  if (!isSelectionComplete()) { toast('Select four distinct players', 'error'); return; }
  let scores;
  try { scores = getMatchScores(); } catch (error) { toast(error.message, 'error'); return; }
  const teamA = [state.selected.A0, state.selected.A1];
  const teamB = [state.selected.B0, state.selected.B1];
  if (scores.sets) {
    els.confirmContent.innerHTML = `
      <div class="confirm-teams">${escapeHtml(teamA.join(' & '))} vs ${escapeHtml(teamB.join(' & '))}</div>
      ${scores.sets.map((match, index) => `<div class="confirm-score">Match ${index + 1}: ${match.team1}–${match.team2}</div>`).join('')}
    `;
    openSheet(els.confirmSheet, els.recordBtn);
    return;
  }
  if (scores.score1 === scores.score2) { toast('Tied scores cannot be submitted', 'error'); return; }
  const a = scores.score1, b = scores.score2;
  const winners = a > b ? teamA : teamB;
  const losers = a > b ? teamB : teamA;
  const winnerScore = Math.max(a,b), loserScore = Math.min(a,b);
  els.confirmContent.innerHTML = `
    <div class="confirm-teams">${escapeHtml(winners.join(' & '))}</div>
    <div class="confirm-verdict">defeated</div>
    <div class="confirm-teams">${escapeHtml(losers.join(' & '))}</div>
    <div class="confirm-score">${winnerScore}–${loserScore}</div>
  `;
  openSheet(els.confirmSheet, els.recordBtn);
}
async function submitMatch() {
  if (submitting) return;
  let scores;
  try { scores = getMatchScores(); } catch (error) { toast(error.message, 'error'); return; }
  submitting = true;
  els.confirmBtn.disabled = true;
  els.confirmBtn.textContent = 'Recording…';
  const teamA = [state.selected.A0, state.selected.A1];
  const teamB = [state.selected.B0, state.selected.B1];
  const payload = { team1: teamA, team2: teamB, score1: scores.score1, score2: scores.score2, ...(scores.sets ? { sets: scores.sets } : {}) };
  try {
    const data = await API.createMatch(payload);
    state.lastResult = data;
    closeSheet(els.confirmSheet);
    renderResult(data);
    toast('Match recorded', 'success');
    await Promise.all([loadRankings(), loadHistory()]);
  } catch (e) {
    if (e.status === 401) { closeSheet(els.confirmSheet); return handleSessionExpired(); }
    toast(e.message || 'Failed to record match', 'error', 3500);
    showRecordError(e.message);
  } finally {
    submitting = false;
    els.confirmBtn.disabled = false;
    els.confirmBtn.textContent = 'Confirm result';
  }
}
function renderResult(data) {
  const match = data.match;
  const recordedMatches = data.matches || [match];
  els.resultSummary.innerHTML = recordedMatches.length > 1
    ? recordedMatches.map((item, index) => `<div>Match ${index + 1}: ${escapeHtml(item.winners.join(' & '))} ${item.winner_score}–${item.loser_score}</div>`).join('')
    : escapeHtml(data.message || `Recorded ${match.winners.join(' & ')} over ${match.losers.join(' & ')} (${match.winner_score}-${match.loser_score}).`);
  els.resultChanges.innerHTML = '';
  const combinedChanges = new Map();
  recordedMatches.forEach(item => (item.changes || []).forEach(ch => {
    const key = ch.name.toLocaleLowerCase();
    const existing = combinedChanges.get(key);
    if (existing) {
      existing.after = ch.after;
      existing.delta = Number(existing.delta) + Number(ch.delta);
    } else combinedChanges.set(key, { ...ch });
  }));
  combinedChanges.forEach(ch => {
    const delta = Number(ch.delta);
    const pos = delta >= 0;
    const div = document.createElement('div');
    div.className = 'result-change';
    div.innerHTML = `
      <span><span class="result-change__name">${escapeHtml(ch.name)}</span><br><span class="result-change__elo">${Number(ch.before).toFixed(2)} → ${Number(ch.after).toFixed(2)}</span></span>
      <span class="result-change__delta ${pos ? 'result-change__delta--positive' : 'result-change__delta--negative'}">${pos ? '+' : ''}${delta.toFixed(2)}</span>
    `;
    els.resultChanges.appendChild(div);
  });
  els.resultPanel.hidden = false;
  els.resultPanel.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}
function clearResultAndReset() {
  els.resultPanel.hidden = true;
  resetSelection();
  syncScoresUI();
  els.matchSets.value = '';
  renderSelectors();
}

// History
async function loadHistory(limit=25) {
  if (state.loading.history) return;
  state.loading.history = true;
  state.error.history = null;
  els.historySkeleton.hidden = false;
  els.historyError.hidden = true;
  els.historyEmpty.hidden = true;
  const btn = els.historyRefresh;
  if (btn) { btn.classList.add('is-loading'); btn.disabled = true; }
  try {
    const data = await API.getMatches(limit);
    state.matches = data.matches || [];
    renderHistory();
    if (btn) { btn.classList.remove('is-loading'); btn.classList.add('is-success'); setTimeout(()=>btn.classList.remove('is-success'), 1500); }
  } catch (e) {
    if (e.status === 401) return handleSessionExpired();
    state.error.history = e.message;
    els.historyError.hidden = false;
    toast(e.message, 'error');
    if (btn) btn.classList.remove('is-loading');
  } finally {
    state.loading.history = false;
    els.historySkeleton.hidden = true;
    if (btn) btn.disabled = false;
  }
}
function formatTimestamp(match) {
  // Try common fields, omit if absent
  const ts = match.created_at || match.timestamp || match.date || null;
  if (!ts) return '';
  try {
    const d = new Date(ts);
    if (isNaN(d)) return '';
    const now = new Date();
    const isToday = d.toDateString() === now.toDateString();
    if (isToday) return `Today · ${d.toLocaleTimeString([], {hour:'numeric', minute:'2-digit'})}`;
    return d.toLocaleDateString([], { month:'short', day:'numeric' }) + ' · ' + d.toLocaleTimeString([], {hour:'numeric', minute:'2-digit'});
  } catch { return ''; }
}
function renderHistory() {
  els.historyList.innerHTML = '';
  if (!state.matches.length) {
    els.historyEmpty.hidden = false;
    return;
  }
  els.historyEmpty.hidden = true;
  // matches already newest first from API
  state.matches.forEach((m, idx) => {
    const card = document.createElement('div');
    card.className = 'history-card';
    const winners = (m.winners || []).join(' & ');
    const losers = (m.losers || []).join(' & ');
    const scoreW = m.winner_score ?? m.score1 ?? '?';
    const scoreL = m.loser_score ?? m.score2 ?? '?';
    const ts = formatTimestamp(m);
    const isLatest = idx === 0;
    card.innerHTML = `
      <div class="history-top">
        <div class="history-teams">
          <div class="history-team">${escapeHtml(winners)}</div>
          <div class="history-team history-team--loser">${escapeHtml(losers)}</div>
          ${ts ? `<div class="history-meta"><span>${escapeHtml(ts)}</span></div>` : ''}
        </div>
        <div class="history-score">
          <div class="history-score__line">${escapeHtml(String(scoreW))}</div>
          <div class="history-score__line history-score__line--loser">${escapeHtml(String(scoreL))}</div>
        </div>
      </div>
      <div class="history-actions">
        <button class="link-btn" data-expand="${escapeHtml(m.id)}">Details</button>
        ${isLatest ? `<button class="link-btn" data-rollback="${escapeHtml(m.id)}" aria-label="Undo latest match">Undo</button>` : ''}
      </div>
      <div class="history-details" id="details-${escapeHtml(m.id)}" hidden>
        ${(m.changes || []).map(ch => {
          const d = Number(ch.delta);
          const cls = d >= 0 ? 'detail-delta--positive' : 'detail-delta--negative';
          return `<div class="history-detail-row"><span>${escapeHtml(ch.name)} ${Number(ch.before).toFixed(2)} → ${Number(ch.after).toFixed(2)}</span><span class="${cls}">${d>=0?'+':''}${d.toFixed(2)}</span></div>`;
        }).join('') || '<span class="text-muted text-sm">No Elo details</span>'}
      </div>
    `;
    els.historyList.appendChild(card);
  });
  // Expand/Collapse and rollback
  els.historyList.querySelectorAll('[data-expand]').forEach(btn => {
    btn.addEventListener('click', () => {
      const id = btn.getAttribute('data-expand');
      const details = document.getElementById('details-'+id);
      const isHidden = details.hidden;
      details.hidden = !isHidden;
      btn.textContent = isHidden ? 'Hide' : 'Details';
    });
  });
  els.historyList.querySelectorAll('[data-rollback]').forEach(btn => {
    btn.addEventListener('click', () => {
      const id = btn.getAttribute('data-rollback');
      const m = state.matches.find(x=>x.id===id);
      if (!m) return;
      pendingRollbackId = id;
      els.rollbackContent.innerHTML = `
        <div class="confirm-teams">${escapeHtml((m.winners||[]).join(' & '))} ${escapeHtml(String(m.winner_score))}</div>
        <div class="confirm-teams confirm-teams--secondary">${escapeHtml((m.losers||[]).join(' & '))} ${escapeHtml(String(m.loser_score))}</div>
      `;
      openSheet(els.rollbackSheet, btn);
    });
  });
}
async function doRollback() {
  if (state.loading.rollback) return;
  state.loading.rollback = true;
  els.rollbackConfirmBtn.disabled = true;
  els.rollbackConfirmBtn.textContent = 'Undoing…';
  try {
    const data = await API.rollback();
    pendingRollbackId = null;
    closeSheet(els.rollbackSheet);
    toast('Match rolled back', 'success');
    await Promise.all([loadRankings(), loadHistory()]);
  } catch (e) {
    if (e.status === 401) { closeSheet(els.rollbackSheet); return handleSessionExpired(); }
    if (e.status === 404) toast('No match to rollback', 'error');
    else toast(e.message || 'Rollback failed', 'error');
  } finally {
    state.loading.rollback = false;
    els.rollbackConfirmBtn.disabled = false;
    els.rollbackConfirmBtn.textContent = 'Undo match';
  }
}

// Initial data
async function loadInitialData() {
  await Promise.all([loadRankings(), loadHistory(), ensurePlayers(), loadNextMatch()]);
  renderSelectors();
}

function syncScoresUI() {
  els.scoreA.textContent = String(state.scores.a);
  els.scoreB.textContent = String(state.scores.b);
  updateWinnerPreview();
  updateRecordButton();
}

// Events
function setupEvents() {
  // Login
  els.togglePw.addEventListener('click', () => {
    const isPw = els.password.type === 'password';
    els.password.type = isPw ? 'text' : 'password';
    els.togglePw.setAttribute('aria-pressed', String(isPw));
    els.togglePw.setAttribute('aria-label', isPw ? 'Hide password' : 'Show password');
    // swap icon
    const use = els.togglePw.querySelector('use');
    if (use) use.setAttribute('href', isPw ? '/static/icons.svg#eye-off' : '/static/icons.svg#eye');
  });
  els.loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const pw = els.password.value;
    if (!pw) { showLoginError('Password is required'); return; }
    if (state.loading.login) return;
    setLoginLoading(true);
    showLoginError('');
    try {
      const access = await API.login(pw);
      if (access.role !== 'admin') throw new Error('Use the match recording password');
      state.authenticated = true;
      state.role = 'admin';
      renderAuth();
      await ensurePlayers();
      switchView('record');
      toast('Recording unlocked', 'success');
      // never store password
      els.password.value = '';
    } catch (err) {
      if (err.status === 401) showLoginError('Invalid password');
      else if (err.status === 500 && /not configured/i.test(err.message)) showLoginError('Web login is not configured. Set WEB_ADMIN_PASSWORD, then restart the server.');
      else showLoginError(err.message || 'Login failed');
    } finally {
      setLoginLoading(false);
    }
  });
  els.loginCancel.addEventListener('click', closeRecordingLogin);

  // Settings
  els.settingsBtn.addEventListener('click', () => {
    const isHidden = els.settingsMenu.hidden;
    els.settingsMenu.hidden = !isHidden;
    els.settingsBtn.setAttribute('aria-expanded', String(isHidden));
  });
  document.addEventListener('click', (e) => {
    if (!els.settingsMenu.contains(e.target) && !els.settingsBtn.contains(e.target)) {
      els.settingsMenu.hidden = true;
      els.settingsBtn.setAttribute('aria-expanded','false');
    }
  });
  els.refreshData.addEventListener('click', async () => {
    els.settingsMenu.hidden = true;
    els.settingsBtn.setAttribute('aria-expanded','false');
    await Promise.all([loadRankings(), loadHistory(), loadNextMatch()]);
    toast('Data refreshed', 'success');
  });
  els.manageNextMatch.addEventListener('click', () => {
    els.settingsMenu.hidden = true;
    els.settingsBtn.setAttribute('aria-expanded','false');
    openNextMatchEditor(els.manageNextMatch);
  });
  els.logoutBtn.addEventListener('click', async () => {
    els.settingsMenu.hidden = true;
    els.settingsBtn.setAttribute('aria-expanded','false');
    try { await API.logout(); } catch {}
    state.authenticated = false;
    state.role = 'viewer';
    renderAuth();
    switchView('ranking');
    toast('Recording locked', 'success');
  });

  // Nav
  $$('.nav-item').forEach(b => b.addEventListener('click', () => switchView(b.dataset.view)));
  $('#ranking-refresh').addEventListener('click', loadRankings);
  $('#history-refresh').addEventListener('click', () => loadHistory(25));
  $('[data-retry="ranking"]').addEventListener('click', loadRankings);
  $('[data-retry="history"]').addEventListener('click', () => loadHistory(25));

  // Record selectors
  $$('.player-select').forEach(btn => {
    btn.addEventListener('click', () => openPlayerSheet(btn.dataset.team, btn.dataset.slot, btn));
  });
  // Close sheets
  $$('[data-close="sheet"]').forEach(b => b.addEventListener('click', () => closeSheet(els.playerSheet)));
  $$('[data-close="confirm"]').forEach(b => b.addEventListener('click', () => closeSheet(els.confirmSheet)));
  $$('[data-close="rollback"]').forEach(b => b.addEventListener('click', () => closeSheet(els.rollbackSheet)));
  $$('[data-close="players"]').forEach(b => b.addEventListener('click', () => closeSheet(els.playersSheet)));
  $$('[data-close="next-match"]').forEach(b => b.addEventListener('click', () => closeSheet(els.nextMatchSheet)));
  $$('[data-close="profile"]').forEach(b => b.addEventListener('click', () => closeSheet(els.profileSheet)));
  // Score
  $$('.score-btn').forEach(b => {
    b.addEventListener('click', () => {
      const team = b.dataset.team;
      const action = b.dataset.action;
      changeScore(team, action==='inc'?1:-1);
    });
  });
  els.recordBtn.addEventListener('click', showConfirm);
  els.matchSets.addEventListener('input', () => { updateWinnerPreview(); updateRecordButton(); });
  els.managePlayersBtn.addEventListener('click', () => openPlayerManagement(els.managePlayersBtn));
  els.addPlayerForm.addEventListener('submit', async (event) => {
    event.preventDefault();
    const name = els.addPlayerName.value.trim();
    const eloText = els.addPlayerElo.value.trim();
    if (!name) return;
    const elo = eloText === '' ? undefined : Number(eloText);
    if (elo !== undefined && !Number.isFinite(elo)) {
      toast('Enter a valid Elo value', 'error');
      return;
    }
    const submit = els.addPlayerForm.querySelector('button[type="submit"]');
    submit.disabled = true;
    try {
      await API.createPlayer({ name, elo });
      els.addPlayerForm.reset();
      await loadRankings();
      renderPlayerManagement();
      toast(`${name} added`, 'success');
    } catch (error) {
      if (error.status === 401) return handleSessionExpired();
      toast(error.message || 'Could not add player', 'error');
    } finally {
      submit.disabled = false;
    }
  });
  els.nextMatchForm.addEventListener('submit', async (event) => {
    event.preventDefault();
    const date = els.nextMatchInputDate.value;
    const time = els.nextMatchInputTime.value;
    const location = els.nextMatchInputLocation.value.trim();
    if (!date || !time || !location) {
      els.nextMatchFormError.textContent = 'Date, time, and location are required.';
      els.nextMatchFormError.hidden = false;
      return;
    }
    els.nextMatchSave.disabled = true;
    els.nextMatchSave.textContent = 'Saving…';
    els.nextMatchFormError.hidden = true;
    try {
      const data = await API.saveNextMatch({ date, time, location });
      state.nextMatch = data.next_match;
      renderNextMatch();
      closeSheet(els.nextMatchSheet);
      toast('Next match saved', 'success');
    } catch (error) {
      if (error.status === 401) return handleSessionExpired();
      els.nextMatchFormError.textContent = error.message || 'Could not save next match.';
      els.nextMatchFormError.hidden = false;
    } finally {
      els.nextMatchSave.disabled = false;
      els.nextMatchSave.textContent = 'Save match';
    }
  });
  els.nextMatchClear.addEventListener('click', async () => {
    if (!state.nextMatch || !window.confirm('Clear the scheduled next match?')) return;
    els.nextMatchClear.disabled = true;
    try {
      await API.clearNextMatch();
      state.nextMatch = null;
      renderNextMatch();
      closeSheet(els.nextMatchSheet);
      toast('Next match cleared', 'success');
    } catch (error) {
      if (error.status === 401) return handleSessionExpired();
      els.nextMatchFormError.textContent = error.message || 'Could not clear next match.';
      els.nextMatchFormError.hidden = false;
    } finally {
      els.nextMatchClear.disabled = false;
    }
  });
  els.confirmBtn.addEventListener('click', submitMatch);
  els.rollbackConfirmBtn.addEventListener('click', doRollback);
  els.recordReset.addEventListener('click', () => {
    clearResultAndReset();
    toast('Cleared', 'success');
  });
  els.viewRankingBtn.addEventListener('click', () => switchView('ranking'));
  els.recordAnotherBtn.addEventListener('click', clearResultAndReset);
  els.shareResultBtn.addEventListener('click', async () => {
    if (!state.lastResult) return;
    const match = state.lastResult.match;
    const text = `Padelo · ${match.winners.join(' & ')} beat ${match.losers.join(' & ')} ${match.winner_score}–${match.loser_score}`;
    try {
      if (navigator.share) await navigator.share({ title: 'Padelo result', text });
      else { await navigator.clipboard.writeText(text); toast('Result copied to clipboard', 'success'); }
    } catch (error) { if (error.name !== 'AbortError') toast('Could not share result', 'error'); }
  });

  // Global escape already handled via sheet; also close settings on escape
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && !els.settingsMenu.hidden) {
      els.settingsMenu.hidden = true;
      els.settingsBtn.setAttribute('aria-expanded','false');
      els.settingsBtn.focus();
    }
  });
}

// Init
(async function init() {
  if ('serviceWorker' in navigator) navigator.serviceWorker.register('/static/service-worker.js').catch(() => {});
  setupEvents();
  syncScoresUI();
  renderSelectors();
  state.checkingSession = false;
  renderAuth();
  checkSession();
  await new Promise(resolve => setTimeout(resolve, 2500));
  els.loadingScreen.hidden = true;
  els.loadingScreen.setAttribute('aria-hidden', 'true');
  // Ensure record button state
  updateRecordButton();
})();
