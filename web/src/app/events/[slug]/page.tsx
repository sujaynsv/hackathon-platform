import Link from 'next/link';
import { api } from '@/lib/api';
import type { EventDetail } from '@/types/events';
import styles from './detail.module.css';

interface Props {
  params: {
    slug: string;
  };
}

export default async function EventDetailPage({ params }: Props) {
  let event: EventDetail | null = null;
  let error: string | null = null;

  try {
    const res = await api.get<EventDetail>(`/events/${params.slug}`);
    event = res.data ?? null;
  } catch (err: unknown) {
    const e = err as Error;
    error = e.message || 'Failed to load event';
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.errorState}>
          <h2>Error</h2>
          <p>{error}</p>
          <Link href="/events" className={styles.backLink}>← Back to Events</Link>
        </div>
      </div>
    );
  }

  if (!event) {
    return null;
  }

  return (
    <div className={styles.container}>
      <Link href="/events" className={styles.backLink}>← All Events</Link>
      
      <header className={styles.header}>
        <h1 className={styles.title}>{event.title}</h1>
        <div className={styles.meta}>
          <span className={styles.status}>{event.status.replace('_', ' ')}</span>
          {event.myRole && (
            <span className={styles.role}>Your role: {event.myRole}</span>
          )}
        </div>
      </header>

      <section className={styles.section}>
        <h2>About</h2>
        <p className={styles.description}>{event.description || 'No description provided.'}</p>
      </section>

      <div className={styles.columns}>
        <section className={styles.section}>
          <h2>Tracks</h2>
          {event.tracks && event.tracks.length > 0 ? (
            <ul className={styles.trackList}>
              {event.tracks.map((track) => (
                <li key={track.id} className={styles.trackItem}>
                  <h3>{track.name}</h3>
                  {track.description && <p>{track.description}</p>}
                </li>
              ))}
            </ul>
          ) : (
            <p className={styles.emptyText}>No tracks defined for this event.</p>
          )}
        </section>

        <section className={styles.section}>
          <h2>Details</h2>
          <dl className={styles.detailsList}>
            <dt>Max Team Size</dt>
            <dd>{event.maxTeamSize} members</dd>
            
            {event.registrationOpensAt && (
              <>
                <dt>Registration Opens</dt>
                <dd>{new Date(event.registrationOpensAt).toLocaleDateString()}</dd>
              </>
            )}
            
            {event.submissionDeadlineAt && (
              <>
                <dt>Submission Deadline</dt>
                <dd>{new Date(event.submissionDeadlineAt).toLocaleDateString()}</dd>
              </>
            )}
          </dl>
          
          <div className={styles.actions}>
            {event.status === 'registration_open' && !event.myRole && (
              <button className={styles.primaryButton}>Register Now</button>
            )}
            {event.status === 'submissions_open' && (
               <Link href={`/events/${event.slug}/submissions/new`} className={styles.primaryButton}>
                 Submit Project
               </Link>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
