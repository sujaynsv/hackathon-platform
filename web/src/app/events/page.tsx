'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import type { Event } from '@/types/events';
import styles from './events.module.css';
import { useAuth } from '@/context/AuthContext';

export default function EventsPage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [dataLoading, setDataLoading] = useState(true);
  const { isLoading: authLoading } = useAuth();

  useEffect(() => {
    if (authLoading) return;
    
    api.get<Event[]>('/events?page=1&pageSize=50')
      .then(res => setEvents(res.data ?? []))
      .catch(err => console.error('Failed to load events', err))
      .finally(() => setDataLoading(false));
  }, [authLoading]);

  if (authLoading || dataLoading) return <div className={styles.container}><div className={styles.emptyState}>Loading...</div></div>;

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <h1 className={styles.title}>Hackathons</h1>
        <Link href="/events/new" className={styles.createButton}>
          Create Event
        </Link>
      </header>

      <div className={styles.grid}>
        {events.length === 0 ? (
          <div className={styles.emptyState}>No events found.</div>
        ) : (
          events.map((event) => (
            <Link key={event.id} href={`/events/${event.slug}`} className={styles.card}>
              <div className={styles.cardContent}>
                <h2 className={styles.cardTitle}>{event.title}</h2>
                <div className={styles.cardStatus}>{event.status.replace('_', ' ')}</div>
                <p className={styles.cardDescription}>{event.description || 'No description provided.'}</p>
              </div>
            </Link>
          ))
        )}
      </div>
    </div>
  );
}
