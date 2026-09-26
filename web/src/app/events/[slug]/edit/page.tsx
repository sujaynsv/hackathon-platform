'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import type { EventDetail, UpdateEventRequest } from '@/types/events';
import styles from './edit.module.css';
import Link from 'next/link';

interface Props {
  params: {
    slug: string;
  };
}

export default function EditEventPage({ params }: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(true);
  const [error, setError] = useState('');
  const [event, setEvent] = useState<EventDetail | null>(null);
  
  const [formData, setFormData] = useState<UpdateEventRequest>({});

  useEffect(() => {
    const fetchEvent = async () => {
      try {
        const res = await api.get<EventDetail>(`/events/${params.slug}`);
        if (res.data) {
          if (res.data.myRole !== 'organizer') {
            router.push(`/events/${params.slug}`);
            return;
          }
          setEvent(res.data);
          setFormData({
            title: res.data.title,
            description: res.data.description,
            status: res.data.status,
            maxTeamSize: res.data.maxTeamSize,
          });
        }
      } catch (err: unknown) {
        setError('Failed to load event for editing');
      } finally {
        setFetching(false);
      }
    };
    fetchEvent();
  }, [params.slug, router]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: name === 'maxTeamSize' ? parseInt(value) || 5 : value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const res = await api.patch<EventDetail>(`/events/${params.slug}`, formData);
      if (res.data) {
        router.push(`/events/${res.data.slug}`);
      }
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || 'Failed to update event');
    } finally {
      setLoading(false);
    }
  };

  if (fetching) {
    return <div className={styles.container}>Loading...</div>;
  }

  if (!event) {
    return <div className={styles.container}>Event not found</div>;
  }

  return (
    <div className={styles.container}>
      <Link href={`/events/${params.slug}`} className={styles.backLink}>← Back to Event</Link>
      
      <header className={styles.header}>
        <h1 className={styles.title}>Edit Event</h1>
        <p className={styles.subtitle}>Update metadata and change the event status.</p>
      </header>

      <form onSubmit={handleSubmit} className={styles.form}>
        {error && <div className={styles.errorAlert}>{error}</div>}
        
        <div className={styles.formGroup}>
          <label htmlFor="title">Event Title *</label>
          <input
            type="text"
            id="title"
            name="title"
            required
            value={formData.title || ''}
            onChange={handleChange}
            className={styles.input}
          />
        </div>

        <div className={styles.formGroup}>
          <label htmlFor="status">Event Status</label>
          <select
            id="status"
            name="status"
            value={formData.status || event.status}
            onChange={handleChange}
            className={styles.select}
          >
            <option value="draft">Draft</option>
            <option value="registration_open">Registration Open</option>
            <option value="submissions_open">Submissions Open</option>
            <option value="judging">Judging</option>
            <option value="voting">Voting</option>
            <option value="results_published">Results Published</option>
            <option value="archived">Archived</option>
          </select>
          <span className={styles.helpText}>Transitioning to invalid status will fail validation.</span>
        </div>

        <div className={styles.formGroup}>
          <label htmlFor="maxTeamSize">Max Team Size</label>
          <input
            type="number"
            id="maxTeamSize"
            name="maxTeamSize"
            min="1"
            max="20"
            value={formData.maxTeamSize || ''}
            onChange={handleChange}
            className={styles.input}
            disabled={event.status !== 'draft' && event.status !== 'registration_open'}
          />
          {(event.status !== 'draft' && event.status !== 'registration_open') && (
            <span className={styles.helpText}>Cannot be changed after registration closes.</span>
          )}
        </div>

        <div className={styles.formGroup}>
          <label htmlFor="description">Description</label>
          <textarea
            id="description"
            name="description"
            rows={5}
            value={formData.description || ''}
            onChange={handleChange}
            className={styles.textarea}
          />
        </div>

        <button type="submit" disabled={loading} className={styles.submitButton}>
          {loading ? 'Saving...' : 'Save Changes'}
        </button>
      </form>
    </div>
  );
}
