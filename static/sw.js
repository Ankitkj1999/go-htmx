
const CACHE_NAME = 'quiz-app-v1';
const OFFLINE_URL = '/offline.html';
const ASSETS_TO_CACHE = [
  '/',
  '/offline.html',
  'https://unpkg.com/htmx.org@1.9.10',
  'https://cdn.tailwindcss.com',
  '/manifest.json'
];

// Install event - cache basic assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(ASSETS_TO_CACHE))
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keyList) => {
      return Promise.all(
        keyList.map((key) => {
          if (key !== CACHE_NAME) {
            return caches.delete(key);
          }
        })
      );
    })
  );
});

// Fetch event - handle offline functionality
self.addEventListener('fetch', (event) => {
  // Handle API requests differently from static assets
  if (event.request.url.includes('/api/') || 
      event.request.url.includes('/submit-question') || 
      event.request.url.includes('/check-answer')) {
    event.respondWith(
      fetch(event.request)
        .catch(() => {
          // If offline, store the request in IndexedDB for later sync
          return storeRequestForLater(event.request)
            .then(() => {
              return new Response(
                JSON.stringify({ 
                  error: 'You are offline. Your request will be processed when you are back online.' 
                }),
                { 
                  headers: { 'Content-Type': 'application/json' }
                }
              );
            });
        })
    );
  } else {
    // For non-API requests, try cache first, then network
    event.respondWith(
      caches.match(event.request)
        .then((response) => {
          return response || fetch(event.request)
            .then((response) => {
              return caches.open(CACHE_NAME)
                .then((cache) => {
                  cache.put(event.request, response.clone());
                  return response;
                });
            })
            .catch(() => {
              return caches.match(OFFLINE_URL);
            });
        })
    );
  }
});

// Background sync
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-questions') {
    event.waitUntil(syncQuestions());
  }
});

// IndexedDB setup for offline data
const dbName = 'QuizOfflineDB';
const storeName = 'offlineQuestions';

function openDB() {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(dbName, 1);
    
    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
    
    request.onupgradeneeded = (event) => {
      const db = event.target.result;
      if (!db.objectStoreNames.contains(storeName)) {
        db.createObjectStore(storeName, { keyPath: 'id', autoIncrement: true });
      }
    };
  });
}

async function storeRequestForLater(request) {
  const db = await openDB();
  const tx = db.transaction(storeName, 'readwrite');
  const store = tx.objectStore(storeName);
  
  const serializedRequest = {
    url: request.url,
    method: request.method,
    headers: Array.from(request.headers.entries()),
    body: await request.clone().text()
  };
  
  return store.add(serializedRequest);
}

async function syncQuestions() {
  const db = await openDB();
  const tx = db.transaction(storeName, 'readwrite');
  const store = tx.objectStore(storeName);
  
  const requests = await store.getAll();
  
  for (const request of requests) {
    try {
      await fetch(request.url, {
        method: request.method,
        headers: new Headers(request.headers),
        body: request.body
      });
      
      await store.delete(request.id);
    } catch (error) {
      console.error('Sync failed for request:', error);
    }
  }
}