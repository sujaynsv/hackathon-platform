'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { Check, LoaderCircle, Pencil, UserRoundMinus, UserRoundPlus } from 'lucide-react';
import { useAuth } from '@/context/AuthContext';
import { api, ApiClientError } from '@/lib/api';
import type { EventDetail } from '@/types/events';
import styles from './detail.module.css';

export function EventActions({ event }: { event: EventDetail }) {
  const { isAuthenticated, isLoading, user } = useAuth();
  const router = useRouter();
  const isOrganizer = isAuthenticated && (
    event.myRole === 'organizer' || user?.id === event.organizerId
  );
  const [myRole, setMyRole] = useState(event.myRole ?? null);
  const [roleResolved, setRoleResolved] = useState(!isAuthenticated || event.myRole != null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const isRegistered = myRole === 'participant';

  useEffect(() => {
    if (!isAuthenticated) {
      setMyRole(event.myRole ?? null);
      setRoleResolved(true);
      return;
    }

    let cancelled = false;
    setRoleResolved(false);
    api.get<EventDetail>(`/events/${encodeURIComponent(event.slug)}`)
      .then((res) => {
        if (!cancelled) setMyRole(res.data.myRole ?? null);
      })
      .catch(() => {
        if (!cancelled) setMyRole(event.myRole ?? null);
      })
      .finally(() => {
        if (!cancelled) setRoleResolved(true);
      });

    return () => {
      cancelled = true;
    };
  }, [event.myRole, event.slug, isAuthenticated]);

  if (isLoading) {
    return <div className={styles.actions}>Checking status...</div>;
  }

  const handleAuthRedirect = () => {
    router.push(`/login`);
  };

  const registrationPath = `/events/${encodeURIComponent(event.slug)}/register`;

  const register = async () => {
    setIsSubmitting(true);
    setError(null);
    setNotice(null);

    try {
      await api.post<{ registered: boolean }>(registrationPath, {});
      setMyRole('participant');
      setNotice('You are registered for this event.');
    } catch (err) {
      if (err instanceof ApiClientError) {
        if (err.code === 'DUPLICATE_RESOURCE') {
          setMyRole('participant');
          setNotice('You are already registered for this event.');
        } else if (err.code === 'DEADLINE_PASSED') {
          setError('Registration has closed.');
        } else if (err.code === 'INVARIANT_VIOLATION') {
          setError(err.message || 'Registration is not available for this event.');
        } else {
          setError(err.message);
        }
      } else {
        setError('Could not register right now. Try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const unregister = async () => {
    setIsSubmitting(true);
    setError(null);
    setNotice(null);

    try {
      await api.delete<{ unregistered: boolean }>(registrationPath);
      setMyRole(null);
      setNotice('You are no longer registered for this event.');
    } catch (err) {
      if (err instanceof ApiClientError) {
        if (err.code === 'INVARIANT_VIOLATION') {
          setError('You cannot unregister after joining a team.');
        } else if (err.code === 'NOT_FOUND') {
          setMyRole(null);
          setError('You are not registered for this event.');
        } else {
          setError(err.message);
        }
      } else {
        setError('Could not unregister right now. Try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className={styles.actions}>
      {isOrganizer && (
        <Link href={`/events/${event.slug}/edit`} className={styles.secondaryButton}>
          <Pencil className={styles.actionIcon} aria-hidden="true" />
          Edit Event
        </Link>
      )}

      {(isRegistered || (event.status === 'registration_open' && !myRole && roleResolved)) && (
        isAuthenticated ? (
          isRegistered ? (
            <>
              {notice && (
                <p className={`${styles.actionMessage} ${styles.actionSuccess}`} role="status">
                  <Check className={styles.actionIcon} aria-hidden="true" />
                  {notice}
                </p>
              )}
              <button
                type="button"
                className={styles.secondaryButton}
                onClick={unregister}
                disabled={isSubmitting}
              >
                {isSubmitting ? (
                  <LoaderCircle className={`${styles.actionIcon} ${styles.spinningIcon}`} aria-hidden="true" />
                ) : (
                  <UserRoundMinus className={styles.actionIcon} aria-hidden="true" />
                )}
                {isSubmitting ? 'Unregistering...' : 'Unregister'}
              </button>
            </>
          ) : (
            <button
              type="button"
              className={styles.primaryButton}
              onClick={register}
              disabled={isSubmitting}
            >
              {isSubmitting ? (
                <LoaderCircle className={`${styles.actionIcon} ${styles.spinningIcon}`} aria-hidden="true" />
              ) : (
                <UserRoundPlus className={styles.actionIcon} aria-hidden="true" />
              )}
              {isSubmitting ? 'Registering...' : 'Register Now'}
            </button>
          )
        ) : (
          <div className={styles.authRequired}>
            <p className={styles.authNotice}>You must be logged in to register.</p>
            <button onClick={handleAuthRedirect} className={styles.secondaryButton}>
              Log in to Register
            </button>
          </div>
        )
      )}

      {error && <p className={`${styles.actionMessage} ${styles.actionError}`} role="alert">{error}</p>}
      {notice && !isRegistered && <p className={styles.actionMessage} role="status">{notice}</p>}
      
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
