const CACHE_NAME = 'web-chat-v2';

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
  let data = { title: 'Nova mensagem', body: 'Você recebeu uma mensagem', url: '/app/messages' };
  try {
    data = { ...data, ...(event.data?.json?.() || {}) };
  } catch (_) {}

  event.waitUntil(self.registration.showNotification(data.title, {
    body: data.body,
    data: { url: data.url || '/app/messages' },
    badge: '/icons/logo.png',
    icon: '/icons/logo.png'
  }));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const targetUrl = event.notification?.data?.url || '/app/messages';

  event.waitUntil((async () => {
    const windows = await clients.matchAll({ type: 'window', includeUncontrolled: true });
    for (const win of windows) {
      if (win.url.includes('/app/messages')) {
        await win.focus();
        win.navigate(targetUrl);
        return;
      }
    }
    await clients.openWindow(targetUrl);
  })());
});
