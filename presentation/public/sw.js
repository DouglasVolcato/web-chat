const CACHE_NAME = 'web-chat-v5';

const scopeUrl = new URL(self.registration.scope);
const basePath = scopeUrl.pathname.endsWith('/') ? scopeUrl.pathname.slice(0, -1) : scopeUrl.pathname;
const appHomePath = `${basePath}/app/messages`;

function buildScopedUrl(rawUrl) {
  if (!rawUrl) return appHomePath;

  const candidate = String(rawUrl);

  if (/^https?:\/\//i.test(candidate)) {
    return candidate;
  }

  if (candidate.startsWith(basePath + '/')) {
    return candidate;
  }

  if (candidate.startsWith('/')) {
    return `${basePath}${candidate}`;
  }

  return `${basePath}/${candidate.replace(/^\/+/, '')}`;
}

self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    const cacheNames = await caches.keys();
    await Promise.all(
      cacheNames
        .filter((name) => name.startsWith('web-chat-') && name !== CACHE_NAME)
        .map((name) => caches.delete(name))
    );
    await clients.claim();
  })());
});

self.addEventListener('fetch', (event) => {
  const { request } = event;

  if (request.method !== 'GET') {
    return;
  }

  const requestUrl = new URL(request.url);
  const requestPath = requestUrl.pathname;
  const isSameOrigin = requestUrl.origin === self.location.origin;
  const isMessageStream = request.headers.get('accept')?.includes('text/event-stream') || requestPath === `${basePath}/app/messages/stream`;
  const isStaticAsset =
    isSameOrigin &&
    (
      requestPath.startsWith(`${basePath}/css/`) ||
      requestPath.startsWith(`${basePath}/js/`) ||
      requestPath.startsWith(`${basePath}/icons/`)
    );

  if (isMessageStream) {
    event.respondWith(fetch(request));
    return;
  }

  if (isStaticAsset) {
    event.respondWith(cacheFirst(request));
    return;
  }

  event.respondWith(fetch(request));
});

async function cacheFirst(request) {
  const cache = await caches.open(CACHE_NAME);
  const cached = await cache.match(request, { ignoreSearch: false });
  if (cached) {
    return cached;
  }

  const response = await fetch(request);
  if (response && response.ok) {
    cache.put(request, response.clone()).catch(() => {});
  }
  return response;
}

self.addEventListener('push', (event) => {
  let data = {
    title: 'Nova mensagem',
    body: 'Abra o app para ler sua nova mensagem.',
    chat_id: '',
    url: appHomePath
  };

  try {
    data = { ...data, ...(event.data?.json?.() || {}) };
  } catch (_) {}

  const notificationUrl = buildScopedUrl(data.url || appHomePath);
  const iconUrl = new URL('icons/logo.png', self.registration.scope).href;

  event.waitUntil(self.registration.showNotification(data.title, {
    body: data.body,
    data: { url: notificationUrl, chat_id: data.chat_id || '' },
    badge: iconUrl,
    icon: iconUrl
  }));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const targetUrl = buildScopedUrl(event.notification?.data?.url || appHomePath);

  event.waitUntil((async () => {
    const windows = await clients.matchAll({ type: 'window', includeUncontrolled: true });

    for (const win of windows) {
      const winUrl = new URL(win.url);
      if (winUrl.pathname.startsWith(`${basePath}/`)) {
        await win.focus();
        await win.navigate(targetUrl);
        return;
      }
    }

    await clients.openWindow(targetUrl);
  })());
});
