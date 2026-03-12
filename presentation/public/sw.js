// Minimal Service Worker for PWA Installation Support
const CACHE_NAME = 'super-fala-v1';

self.addEventListener('install', (event) => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(clients.claim());
});

self.addEventListener('fetch', (event) => {
  // Basic pass-through for now. 
  // Requirement for PWA installation is just having a fetch handler.
  event.respondWith(fetch(event.request));
});
