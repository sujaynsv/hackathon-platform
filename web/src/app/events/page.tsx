'use client';

import { useState, useMemo, useEffect } from 'react';
import { apiClient } from '@/lib/api';
import { EventCard } from '@/components/events/EventCard';
import styles from './events.module.css';
import type { EventDTO } from '@/types/api';

type Tab = 'all' | 'open' | 'judging' | 'voting' | 'closed';

export default function EventsPage() {
  const [search, setSearch] = useState('');
  const [activeTab, setActiveTab] = useState<Tab>('all');
  const [events, setEvents] = useState<EventDTO[] | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    apiClient.listEvents()
      .then(setEvents)
      .catch(setError)
      .finally(() => setIsLoading(false));
  }, []);

  const filteredEvents = useMemo(() => {
    if (!events) return [];
    
    return events.filter(event => {
      // Search filter
      const matchesSearch = event.title.toLowerCase().includes(search.toLowerCase());
      if (!matchesSearch) return false;

      // Status filter
      if (activeTab === 'all') return event.status !== 'draft';
      if (activeTab === 'open') return ['registration_open', 'submissions_open'].includes(event.status);
      if (activeTab === 'judging') return event.status === 'judging';
      if (activeTab === 'voting') return event.status === 'voting';
      if (activeTab === 'closed') return ['results_published', 'archived'].includes(event.status);
      
      return true;
    });
  }, [events, search, activeTab]);

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div>
          <h1 className={styles.title}>Hackathons</h1>
          <p className={styles.subtitle}>Discover and join the next big event.</p>
        </div>

        <div className={styles.controls}>
          <div className={styles.tabs}>
            <button 
              className={`${styles.tab} ${activeTab === 'all' ? styles.tabActive : ''}`}
              onClick={() => setActiveTab('all')}
            >
              All Events
            </button>
            <button 
              className={`${styles.tab} ${activeTab === 'open' ? styles.tabActive : ''}`}
              onClick={() => setActiveTab('open')}
            >
              Open
            </button>
            <button 
              className={`${styles.tab} ${activeTab === 'judging' ? styles.tabActive : ''}`}
              onClick={() => setActiveTab('judging')}
            >
              Judging
            </button>
            <button 
              className={`${styles.tab} ${activeTab === 'voting' ? styles.tabActive : ''}`}
              onClick={() => setActiveTab('voting')}
            >
              Voting
            </button>
            <button 
              className={`${styles.tab} ${activeTab === 'closed' ? styles.tabActive : ''}`}
              onClick={() => setActiveTab('closed')}
            >
              Closed
            </button>
          </div>

          <input 
            type="text" 
            placeholder="Search events..." 
            className={styles.searchInput}
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>
      </div>

      {error ? (
        <div className={styles.emptyState} style={{ color: 'var(--error)' }}>
          <div className={styles.emptyIcon}>⚠️</div>
          <h2 className={styles.emptyTitle}>Failed to load events</h2>
          <p className={styles.emptyDesc}>{(error as Error).message || 'An unexpected error occurred.'}</p>
        </div>
      ) : isLoading ? (
        <div className={styles.grid}>
          {[1, 2, 3, 4, 5, 6].map(i => (
            <div key={i} className={styles.skeletonCard}>
              <div className={styles.skeletonBanner} />
              <div className={styles.skeletonContent}>
                <div className={styles.skeletonTitle} />
                <div className={styles.skeletonDesc} />
                <div className={styles.skeletonMeta} />
              </div>
            </div>
          ))}
        </div>
      ) : filteredEvents.length === 0 ? (
        <div className={styles.emptyState}>
          <div className={styles.emptyIcon}>🚀</div>
          <h2 className={styles.emptyTitle}>No events found</h2>
          <p className={styles.emptyDesc}>
            {search ? 'Try adjusting your search or filters.' : 'There are no active hackathons right now.'}
          </p>
        </div>
      ) : (
        <div className={styles.grid}>
          {filteredEvents.map(event => (
            <EventCard key={event.id} event={event} />
          ))}
        </div>
      )}
    </div>
  );
}
