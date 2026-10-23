import './ui-enhancements.js';

export const API = (() => {
  async function request(path, opts = {}) {
    const url = path;
    const fetchOpts = {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
      ...opts,
    };
    if (fetchOpts.body && typeof fetchOpts.body !== 'string') {
      fetchOpts.body = JSON.stringify(fetchOpts.body);
    }
    let res;
    try {
      res = await fetch(url, fetchOpts);
    } catch (e) {
      throw new Error('Network unavailable. Check connection.');
    }
    let data = null;
    const ct = res.headers.get('content-type') || '';
    if (ct.includes('application/json')) {
      try { data = await res.json(); } catch { data = null; }
    } else {
      try { const t = await res.text(); data = t ? { detail: t } : null; } catch { data = null; }
    }
    if (!res.ok) {
      const msg = (data && (data.detail || data.message)) || res.statusText || 'Request failed';
      const err = new Error(msg);
      err.status = res.status;
      err.data = data;
      throw err;
    }
    return data;
  }

  return {
    getSession: () => request('/api/v1/session'),
    getSessionRole: () => request('/api/v1/session/role'),
    login: (password) => request('/api/v1/session', { method: 'POST', body: { password } }),
    logout: () => request('/api/v1/session', { method: 'DELETE' }),
    getRankings: () => request('/api/v1/rankings'),
    getPlayers: () => request('/api/v1/players'),
    createPlayer: ({ name, elo }) => request('/api/v1/players', { method: 'POST', body: { name, ...(elo === undefined ? {} : { elo }) } }),
    deletePlayer: (name) => request(`/api/v1/players/${encodeURIComponent(name)}`, { method: 'DELETE' }),
    renamePlayer: (name, newName) => request(`/api/v1/players/${encodeURIComponent(name)}`, { method: 'PATCH', body: { name: newName } }),
    adjustPlayer: (name, delta) => request(`/api/v1/players/${encodeURIComponent(name)}/elo`, { method: 'PATCH', body: { delta } }),
    getPlayerProfile: (name) => request(`/api/v1/players/${encodeURIComponent(name)}/profile`),
    createMatch: ({ team1, team2, score1, score2, sets }) => request('/api/v1/matches', { method: 'POST', body: { team1, team2, score1, score2, ...(sets ? { sets } : {}) } }),
    getMatches: (limit = 25) => request(`/api/v1/matches?limit=${encodeURIComponent(limit)}`),
    rollback: () => request('/api/v1/matches/latest/rollback', { method: 'POST' }),
    getNextMatch: () => request('/api/v1/next-match'),
    saveNextMatch: ({ date, time, location }) => request('/api/v1/next-match', { method: 'PUT', body: { date, time, location } }),
    clearNextMatch: () => request('/api/v1/next-match', { method: 'DELETE' }),
  };
})();
