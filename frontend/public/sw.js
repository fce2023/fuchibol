self.addEventListener('install', (e) => {
  self.skipWaiting();
});

self.addEventListener('activate', (e) => {
  e.waitUntil(self.clients.claim());
});

self.addEventListener('fetch', (e) => {
  // Pass-through fetch handler is enough to trigger the PWA install prompt in Chrome
  e.respondWith(fetch(e.request).catch(() => {
    // Return offline fallback if needed
  }));
});
