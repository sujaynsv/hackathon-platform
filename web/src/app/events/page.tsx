import Link from 'next/link';
import { api } from '@/lib/api';
import type { Event } from '@/types/events';
import styles from './events.module.css';



export default async function EventsPage() {
  let events: Event[] = [];
  try {
    const res = await api.get<Event[]>('/events?page=1&pageSize=50');
    events = res.data ?? [];
  } catch (err) {
    console.error('Failed to load events', err);
  }

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
