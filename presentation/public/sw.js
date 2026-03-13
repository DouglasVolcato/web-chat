const CACHE_NAME = 'web-chat-v4';

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
  event.waitUntil(clients.claim());
});

self.addEventListener('fetch', (event) => {
  event.respondWith(fetch(event.request));
});

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
