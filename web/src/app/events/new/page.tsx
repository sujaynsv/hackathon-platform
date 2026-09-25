'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import type { CreateEventRequest, Event } from '@/types/events';
import styles from './new.module.css';
import Link from 'next/link';

export default function CreateEventPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  
  const [formData, setFormData] = useState<CreateEventRequest>({
    slug: '',
    title: '',
    description: '',
    maxTeamSize: 5,
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: name === 'maxTeamSize' ? parseInt(value) || 5 : value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      // Clean up empty strings for optional dates if we add them later, 
      // for now we just submit the basics
      const res = await api.post<Event>('/events', formData);
      if (res.data) {
        router.push(`/events/${res.data.slug}`);
      }
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || 'Failed to create event');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.container}>
      <Link href="/events" className={styles.backLink}>← Back to Events</Link>
      
      <header className={styles.header}>
        <h1 className={styles.title}>Create New Event</h1>
        <p className={styles.subtitle}>Start a new hackathon. It will be created as a draft.</p>
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
            value={formData.title}
            onChange={handleChange}
            placeholder="e.g. Dogfood Hackathon 2026"
            className={styles.input}
          />
        </div>

        <div className={styles.formGroup}>
          <label htmlFor="slug">URL Slug *</label>
          <input
            type="text"
            id="slug"
            name="slug"
            required
            value={formData.slug}
            onChange={handleChange}
            placeholder="e.g. dogfood-2026"
            className={styles.input}
            pattern="^[a-z0-9-]+$"
            title="Only lowercase letters, numbers, and hyphens are allowed"
          />
          <span className={styles.helpText}>This will be used in the URL: /events/your-slug</span>
        </div>

        <div className={styles.formGroup}>
          <label htmlFor="maxTeamSize">Max Team Size</label>
          <input
            type="number"
            id="maxTeamSize"
            name="maxTeamSize"
            min="1"
            max="20"
            value={formData.maxTeamSize}
            onChange={handleChange}
            className={styles.input}
          />
        </div>

        <div className={styles.formGroup}>
          <label htmlFor="description">Description</label>
          <textarea
            id="description"
            name="description"
            rows={5}
            value={formData.description}
            onChange={handleChange}
            placeholder="What is this event about?"
            className={styles.textarea}
          />
        </div>

        <button type="submit" disabled={loading} className={styles.submitButton}>
          {loading ? 'Creating...' : 'Create Event'}
        </button>
      </form>
    </div>
  );
}
