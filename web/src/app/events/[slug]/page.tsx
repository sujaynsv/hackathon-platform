'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { apiClient } from '@/lib/api';
import { StatusBadge } from '@/components/events/StatusBadge';
import { DateTimeline } from '@/components/events/DateTimeline';
import styles from './detail.module.css';
import type { EventDetailDTO } from '@/types/api';

export default function EventDetailPage() {
  const params = useParams();
  const slug = params.slug as string;

  const [event, setEvent] = useState<EventDetailDTO | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    apiClient.getEvent(slug)
      .then(setEvent)
      .catch(setError)
      .finally(() => setIsLoading(false));
  }, [slug]);

  if (isLoading) {
    return (
      <div className={styles.container}>
        <div style={{ padding: '4rem', textAlign: 'center', color: 'var(--text-secondary)' }}>
          Loading event details...
        </div>
      </div>
    );
  }

  if (error || !event) {
    return (
      <div className={styles.container}>
        <div className={styles.errorState}>
          <h2>Failed to load event</h2>
          <p>{(error as Error)?.message || 'Event not found'}</p>
          <Link href="/events" className={styles.secondaryBtn} style={{ maxWidth: '200px', margin: '2rem auto' }}>
            Back to Events
          </Link>
        </div>
      </div>
    );
  }

  const isRegistrationOpen = event.status === 'registration_open';
  const showRegistrationCTA = isRegistrationOpen && !event.myRole;

  return (
    <div>
      <div className={styles.hero}>
        {event.bannerUrl ? (
          <img src={event.bannerUrl} alt={event.title} className={styles.heroImg} />
        ) : (
          <div className={styles.heroGradient} />
        )}
        <div className={styles.heroOverlay}>
          <h1 className={styles.title}>{event.title}</h1>
          <div className={styles.metaRow}>
            <StatusBadge status={event.status} />
            {event.myRole && (
              <span className={styles.roleBadge}>
                You are a {event.myRole.charAt(0).toUpperCase() + event.myRole.slice(1)}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className={styles.container}>
        <div className={styles.mainContent}>
          <Link href="/events" className={styles.backLink}>← Back to all events</Link>
          
          <section className={styles.section}>
            <h2 className={styles.sectionTitle}>About this Hackathon</h2>
            <div className={styles.description}>
              {event.description || 'No description provided for this event yet.'}
            </div>
          </section>

          <section className={styles.section}>
            <h2 className={styles.sectionTitle}>Tracks & Prizes</h2>
            {event.tracks && event.tracks.length > 0 ? (
              <div className={styles.trackGrid}>
                {event.tracks.map((track) => (
                  <div key={track.id} className={styles.trackCard}>
                    <h3 className={styles.trackTitle}>{track.name}</h3>
                    {track.description && <p className={styles.trackDesc}>{track.description}</p>}
                  </div>
                ))}
              </div>
            ) : (
              <p style={{ color: 'var(--text-secondary)' }}>No tracks announced yet.</p>
            )}
          </section>
        </div>

        <div className={styles.sidebar}>
          {showRegistrationCTA && (
            <div className={styles.actionCard}>
              <h3 className={styles.actionTitle}>Join the Hackathon</h3>
              <p className={styles.actionDesc}>
                Registration is open! Form a team of up to {event.maxTeamSize} people.
              </p>
              <Link href={`/events/${event.slug}/participate`} className={styles.primaryBtn}>
                Register Now
              </Link>
            </div>
          )}

          {event.myRole === 'organizer' && (
            <div className={styles.actionCard}>
              <h3 className={styles.actionTitle}>Organizer Tools</h3>
              <p className={styles.actionDesc}>Manage this event's settings and lifecycle.</p>
              <Link href={`/events/${event.slug}/manage`} className={styles.primaryBtn}>
                Dashboard
              </Link>
            </div>
          )}

          {event.myRole === 'participant' && (
            <div className={styles.actionCard}>
              <h3 className={styles.actionTitle}>Participant Hub</h3>
              <p className={styles.actionDesc}>Manage your team and submission.</p>
              <Link href={`/events/${event.slug}/participate`} className={styles.primaryBtn}>
                Participate / My Team
              </Link>
              <Link href={`/events/${event.slug}/submissions/new`} className={styles.secondaryBtn}>
                Submit Project
              </Link>
            </div>
          )}

          <div className={styles.actionCard}>
            <Link href={`/events/${event.slug}/gallery`} className={styles.secondaryBtn}>
              View Project Gallery
            </Link>
          </div>

          <div className={styles.statGrid}>
            <div className={styles.statCard}>
              <div className={styles.statValue}>{event.maxTeamSize}</div>
              <div className={styles.statLabel}>Max Team Size</div>
            </div>
            {/* These stats can be populated later with actual counts */}
            <div className={styles.statCard}>
              <div className={styles.statValue}>-</div>
              <div className={styles.statLabel}>Participants</div>
            </div>
          </div>

          <DateTimeline event={event} />
        </div>
      </div>
    </div>
  );
}
