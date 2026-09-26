'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api, apiClient } from '@/lib/api';
import type { EventDetailDTO, EventStatus } from '@/types/api';
import styles from './manage.module.css';

type Panel = 'status' | 'edit' | 'judges' | 'rubric' | 'stats';

export default function ManageEventPage({ params }: { params: { slug: string } }) {
  const router = useRouter();
  const [event, setEvent] = useState<EventDetailDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activePanel, setActivePanel] = useState<Panel>('status');

  // Edit Panel State
  const [editTitle, setEditTitle] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [editMaxTeamSize, setEditMaxTeamSize] = useState(5);
  const [editSaving, setEditSaving] = useState(false);
  const [editSuccess, setEditSuccess] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  useEffect(() => {
    const fetchEvent = async () => {
      try {
        const res = await api.get<EventDetailDTO>(`/events/${params.slug}`);
        if (res.data.myRole !== 'organizer') {
          router.replace(`/events/${params.slug}`);
          return;
        }
        setEvent(res.data);
        setEditTitle(res.data.title);
        setEditDescription(res.data.description || '');
        setEditMaxTeamSize(res.data.maxTeamSize);
      } catch (err: any) {
        setError(err.message || 'Failed to load event');
      } finally {
        setLoading(false);
      }
    };
    fetchEvent();
  }, [params.slug, router]);

  if (loading) {
    return (
      <div className={styles.container}>
        <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-secondary)' }}>
          Loading dashboard...
        </div>
      </div>
    );
  }

  if (error || !event) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>{error || 'Event not found'}</div>
      </div>
    );
  }

  const handleUpdate = async () => {
    setEditSaving(true);
    setEditError(null);
    setEditSuccess(false);
    try {
      const updated = await apiClient.updateEvent(event.slug, {
        title: editTitle !== event.title ? editTitle : undefined,
        description: editDescription !== event.description ? editDescription : undefined,
        maxTeamSize: editMaxTeamSize !== event.maxTeamSize ? editMaxTeamSize : undefined,
      });
      setEvent(updated);
      setEditSuccess(true);
      setTimeout(() => setEditSuccess(false), 3000);
    } catch (err: any) {
      setEditError(err.message || 'Failed to update event');
    } finally {
      setEditSaving(false);
    }
  };

  const handleAdvanceStatus = async (nextStatus: EventStatus) => {
    if (!window.confirm(`Are you sure you want to advance to ${nextStatus}? This cannot be undone.`)) {
      return;
    }
    
    setEditSaving(true);
    try {
      const updated = await apiClient.updateEvent(event.slug, {
        status: nextStatus,
      });
      setEvent(updated);
    } catch (err: any) {
      alert(err.message || 'Failed to advance status');
    } finally {
      setEditSaving(false);
    }
  };

  const getNextStatus = (current: string): { status: EventStatus, label: string } | null => {
    switch (current) {
      case 'draft': return { status: 'registration_open', label: 'Open Registration' };
      case 'registration_open': return { status: 'submissions_open', label: 'Close Registration & Open Submissions' };
      case 'submissions_open': return { status: 'judging', label: 'Start Judging Phase' };
      case 'judging': return { status: 'voting', label: 'Open Public Voting' };
      case 'voting': return { status: 'results_published', label: 'Publish Final Results' };
      default: return null;
    }
  };

  const nextStep = getNextStatus(event.status);
  const isRegistrationClosed = event.status !== 'draft' && event.status !== 'registration_open';

  return (
    <div className={styles.container}>
      <div className={styles.sidebar}>
        <button 
          className={`${styles.navItem} ${activePanel === 'status' ? styles.navItemActive : ''}`}
          onClick={() => setActivePanel('status')}
        >
          Status & Lifecycle
        </button>
        <button 
          className={`${styles.navItem} ${activePanel === 'edit' ? styles.navItemActive : ''}`}
          onClick={() => setActivePanel('edit')}
        >
          Edit Details
        </button>
        <button className={styles.navItem} disabled>Judges (Coming Soon)</button>
        <button className={styles.navItem} disabled>Rubric (Coming Soon)</button>
        <button className={styles.navItem} disabled>Stats (Coming Soon)</button>
      </div>

      <div className={styles.content}>
        {activePanel === 'status' && (
          <div className={styles.panel}>
            <div className={styles.header}>
              <h2 className={styles.title}>Event Lifecycle</h2>
              <p className={styles.subtitle}>Manage the current stage of your hackathon.</p>
            </div>
            
            <div className={styles.statusCard}>
              <div>
                <div className={styles.statusLabel}>Current Status</div>
                <div className={styles.statusValue}>{event.status.replace(/_/g, ' ').toUpperCase()}</div>
              </div>

              {nextStep ? (
                <div style={{ marginTop: '1rem' }}>
                  <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem', fontSize: '0.9rem' }}>
                    Ready to move to the next phase?
                  </p>
                  <button 
                    className={styles.buttonPrimary} 
                    onClick={() => handleAdvanceStatus(nextStep.status)}
                    disabled={editSaving}
                  >
                    {editSaving ? 'Processing...' : nextStep.label}
                  </button>
                </div>
              ) : (
                <div style={{ marginTop: '1rem', color: 'var(--success)' }}>
                  This event has reached the end of its active lifecycle.
                </div>
              )}
            </div>
          </div>
        )}

        {activePanel === 'edit' && (
          <div className={styles.panel}>
            <div className={styles.header}>
              <h2 className={styles.title}>Edit Details</h2>
              <p className={styles.subtitle}>Update your event's public information.</p>
            </div>

            <div className={styles.formGroup}>
              <label className={styles.label}>Event Title</label>
              <input 
                className={styles.input} 
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                placeholder="Awesome Hackathon 2026"
              />
            </div>

            <div className={styles.formGroup}>
              <label className={styles.label}>Description</label>
              <textarea 
                className={styles.textarea} 
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                placeholder="Describe your event..."
              />
            </div>

            <div className={styles.formGroup}>
              <label className={styles.label}>Max Team Size</label>
              <input 
                type="number"
                min="1"
                max="20"
                className={styles.input} 
                value={editMaxTeamSize}
                onChange={(e) => setEditMaxTeamSize(Number(e.target.value))}
                disabled={isRegistrationClosed}
                title={isRegistrationClosed ? 'Cannot change after registration closes' : ''}
              />
              {isRegistrationClosed && (
                <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                  Locked because registration has closed.
                </span>
              )}
            </div>

            {editError && <div className={styles.error}>{editError}</div>}
            {editSuccess && <div className={styles.success}>Changes saved successfully!</div>}

            <button 
              className={styles.buttonPrimary} 
              onClick={handleUpdate}
              disabled={editSaving}
            >
              {editSaving ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
