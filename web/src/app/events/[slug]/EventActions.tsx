'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import type { EventDetail } from '@/types/events';
import styles from './detail.module.css';

export function EventActions({ event }: { event: EventDetail }) {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  if (isLoading) {
    return <div className={styles.actions}>Checking status...</div>;
  }

  const handleAuthRedirect = () => {
    router.push(`/login`);
  };

  return (
    <div className={styles.actions}>
      {event.status === 'registration_open' && !event.myRole && (
        isAuthenticated ? (
          <button className={styles.primaryButton}>Register Now</button>
        ) : (
          <div className={styles.authRequired}>
            <p className={styles.authNotice}>You must be logged in to register.</p>
            <button onClick={handleAuthRedirect} className={styles.secondaryButton}>
              Log in to Register
            </button>
          </div>
        )
      )}
      
      {event.status === 'submissions_open' && (
        isAuthenticated ? (
          <Link href={`/events/${event.slug}/submissions/new`} className={styles.primaryButton}>
            Submit Project
          </Link>
        ) : (
          <div className={styles.authRequired}>
            <p className={styles.authNotice}>You must be logged in to submit a project.</p>
            <button onClick={handleAuthRedirect} className={styles.secondaryButton}>
              Log in to Submit
            </button>
          </div>
        )
      )}
    </div>
  );
}
