const CACHE = 'padelo-public-v2';
const ASSETS = ['/', '/static/styles.css', '/static/app.js', '/static/api.js', '/static/state.js', '/static/ui-enhancements.js', '/static/icons.svg', '/static/manifest.webmanifest',
  '/static/tournament.html', '/static/tournament.css', '/static/tournament.js'];
self.addEventListener('install', event => event.waitUntil(
  caches.open(CACHE).then(cache => cache.addAll(ASSETS)).then(() => self.skipWaiting())
));
self.addEventListener('activate', event => event.waitUntil(
  caches.keys().then(keys => Promise.all(keys.filter(key => key !== CACHE).map(key => caches.delete(key)))).then(() => self.clients.claim())
));
self.addEventListener('fetch', event => {
  if (event.request.method !== 'GET') return;
  const url = new URL(event.request.url);

  // API responses must always come from the server so rankings and history stay current.
  if (url.origin !== self.location.origin || url.pathname.startsWith('/api/')) return;

  const isAppShell = event.request.mode === 'navigate' || url.pathname === '/' || url.pathname.startsWith('/static/');
  if (!isAppShell) return;

  event.respondWith((async () => {
    const cache = await caches.open(CACHE);
    const cacheKey = event.request.mode === 'navigate' ? url.pathname : event.request;
    const cached = await cache.match(cacheKey);
    const refresh = fetch(event.request).then(response => {
      if (response.ok) cache.put(cacheKey, response.clone());
      return response;
    });

    if (cached) {
      event.waitUntil(refresh.catch(() => undefined));
      return cached;
    }
    return refresh;
  })());
});
