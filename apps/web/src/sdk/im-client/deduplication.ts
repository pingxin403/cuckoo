/**
 * IM Client SDK - Deduplication Service
 * 
 * Handles client-side message deduplication using various storage backends.
 */

import type { DeduplicationEntry } from './types';

export interface DeduplicationStorage {
  has(msgId: string): Promise<boolean>;
  add(msgId: string): Promise<void>;
  cleanup(): Promise<void>;
  clear(): Promise<void>;
}

// ============================================================================
// Memory Storage (default, no persistence)
// ============================================================================

export class MemoryDeduplicationStorage implements DeduplicationStorage {
  private entries: Map<string, number> = new Map();
  private ttl: number;

  constructor(ttl: number = 7 * 24 * 60 * 60 * 1000) { // 7 days default
    this.ttl = ttl;
  }

  async has(msgId: string): Promise<boolean> {
    const timestamp = this.entries.get(msgId);
    if (!timestamp) {
return false;
}
    
    // Check if entry has expired
    if (Date.now() - timestamp > this.ttl) {
      this.entries.delete(msgId);
      return false;
    }
    
    return true;
  }

  async add(msgId: string): Promise<void> {
    this.entries.set(msgId, Date.now());
  }

  async cleanup(): Promise<void> {
    const now = Date.now();
    const expiredKeys: string[] = [];
    
    for (const [msgId, timestamp] of this.entries.entries()) {
      if (now - timestamp > this.ttl) {
        expiredKeys.push(msgId);
      }
    }
    
    expiredKeys.forEach(key => this.entries.delete(key));
  }

  async clear(): Promise<void> {
    this.entries.clear();
  }
}

// ============================================================================
// LocalStorage Storage (persists across page reloads)
// ============================================================================

export class LocalStorageDeduplicationStorage implements DeduplicationStorage {
  private storageKey = 'im_client_dedup';
  private ttl: number;

  constructor(ttl: number = 7 * 24 * 60 * 60 * 1000) {
    this.ttl = ttl;
  }

  private getEntries(): Map<string, number> {
    try {
      const data = localStorage.getItem(this.storageKey);
      if (!data) {
return new Map();
}
      
      const entries: DeduplicationEntry[] = JSON.parse(data);
      return new Map(entries.map(e => [e.msg_id, e.timestamp]));
    } catch (error) {
      console.error('Failed to read deduplication entries from localStorage:', error);
      return new Map();
    }
  }

  private saveEntries(entries: Map<string, number>): void {
    try {
      const data: DeduplicationEntry[] = Array.from(entries.entries()).map(([msg_id, timestamp]) => ({
        msg_id,
        timestamp,
      }));
      localStorage.setItem(this.storageKey, JSON.stringify(data));
    } catch (error) {
      console.error('Failed to save deduplication entries to localStorage:', error);
    }
  }

  async has(msgId: string): Promise<boolean> {
    const entries = this.getEntries();
    const timestamp = entries.get(msgId);
    
    if (!timestamp) {
return false;
}
    
    // Check if entry has expired
    if (Date.now() - timestamp > this.ttl) {
      entries.delete(msgId);
      this.saveEntries(entries);
      return false;
    }
    
    return true;
  }

  async add(msgId: string): Promise<void> {
    const entries = this.getEntries();
    entries.set(msgId, Date.now());
    this.saveEntries(entries);
  }

  async cleanup(): Promise<void> {
    const entries = this.getEntries();
    const now = Date.now();
    let hasChanges = false;
    
    for (const [msgId, timestamp] of entries.entries()) {
      if (now - timestamp > this.ttl) {
        entries.delete(msgId);
        hasChanges = true;
      }
    }
    
    if (hasChanges) {
      this.saveEntries(entries);
    }
  }

  async clear(): Promise<void> {
    localStorage.removeItem(this.storageKey);
  }
}

// ============================================================================
// IndexedDB Storage (best for large datasets)
// ============================================================================

export class IndexedDBDeduplicationStorage implements DeduplicationStorage {
  private dbName = 'im_client_db';
  private storeName = 'deduplication';
  private ttl: number;
  private db: IDBDatabase | null = null;

  constructor(ttl: number = 7 * 24 * 60 * 60 * 1000) {
    this.ttl = ttl;
  }

  private async getDB(): Promise<IDBDatabase> {
    if (this.db) {
return this.db;
}

    return new Promise((resolve, reject) => {
      const request = indexedDB.open(this.dbName, 1);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        this.db = request.result;
        resolve(request.result);
      };

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;
        if (!db.objectStoreNames.contains(this.storeName)) {
          const store = db.createObjectStore(this.storeName, { keyPath: 'msg_id' });
          store.createIndex('timestamp', 'timestamp', { unique: false });
        }
      };
    });
  }

  async has(msgId: string): Promise<boolean> {
    const db = await this.getDB();
    
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([this.storeName], 'readonly');
      const store = transaction.objectStore(this.storeName);
      const request = store.get(msgId);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const entry = request.result as DeduplicationEntry | undefined;
        
        if (!entry) {
          resolve(false);
          return;
        }
        
        // Check if entry has expired
        if (Date.now() - entry.timestamp > this.ttl) {
          // Delete expired entry
          const deleteTransaction = db.transaction([this.storeName], 'readwrite');
          const deleteStore = deleteTransaction.objectStore(this.storeName);
          deleteStore.delete(msgId);
          resolve(false);
        } else {
          resolve(true);
        }
      };
    });
  }

  async add(msgId: string): Promise<void> {
    const db = await this.getDB();
    
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([this.storeName], 'readwrite');
      const store = transaction.objectStore(this.storeName);
      const entry: DeduplicationEntry = {
        msg_id: msgId,
        timestamp: Date.now(),
      };
      const request = store.put(entry);

      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }

  async cleanup(): Promise<void> {
    const db = await this.getDB();
    const now = Date.now();
    
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([this.storeName], 'readwrite');
      const store = transaction.objectStore(this.storeName);
      const index = store.index('timestamp');
      const request = index.openCursor();

      request.onerror = () => reject(request.error);
      request.onsuccess = (event) => {
        const cursor = (event.target as IDBRequest).result as IDBCursorWithValue | null;
        
        if (cursor) {
          const entry = cursor.value as DeduplicationEntry;
          if (now - entry.timestamp > this.ttl) {
            cursor.delete();
          }
          cursor.continue();
        } else {
          resolve();
        }
      };
    });
  }

  async clear(): Promise<void> {
    const db = await this.getDB();
    
    return new Promise((resolve, reject) => {
      const transaction = db.transaction([this.storeName], 'readwrite');
      const store = transaction.objectStore(this.storeName);
      const request = store.clear();

      request.onerror = () => reject(request.error);
      request.onsuccess = () => resolve();
    });
  }
}

// ============================================================================
// Factory Function
// ============================================================================

export function createDeduplicationStorage(
  type: 'memory' | 'indexeddb' | 'localstorage',
  ttl: number,
): DeduplicationStorage {
  switch (type) {
    case 'indexeddb':
      return new IndexedDBDeduplicationStorage(ttl);
    case 'localstorage':
      return new LocalStorageDeduplicationStorage(ttl);
    case 'memory':
    default:
      return new MemoryDeduplicationStorage(ttl);
  }
}
