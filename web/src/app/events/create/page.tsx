'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient } from '@/lib/api';
import styles from './create.module.css';

export default function CreateEventPage() {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form State
  const [slug, setSlug] = useState('');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [maxTeamSize, setMaxTeamSize] = useState(5);

  const handleNext = () => setStep(step + 1);
  const handleBack = () => setStep(step - 1);

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);
    try {
      const event = await apiClient.createEvent({
        slug,
        title,
        description,
        maxTeamSize,
        // Dates will default to 30/45 days as per backend defaults
      });
      router.push(`/events/${event.slug}/manage`);
    } catch (err: any) {
      setError(err.message || 'Failed to create event');
      setLoading(false);
    }
  };

  const getStepClass = (s: number) => {
    if (step === s) return `${styles.step} ${styles.stepActive}`;
    if (step > s) return `${styles.step} ${styles.stepCompleted}`;
    return styles.step;
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.title}>Create New Event</h1>
        <p className={styles.subtitle}>Set up your hackathon in just a few steps.</p>
      </div>

      <div className={styles.stepper}>
        <div className={getStepClass(1)}>1</div>
        <div className={getStepClass(2)}>2</div>
        <div className={getStepClass(3)}>3</div>
      </div>

      {error && <div className={styles.error}>{error}</div>}

      {step === 1 && (
        <div>
          <div className={styles.formGroup}>
            <label className={styles.label}>Event Title *</label>
            <input 
              className={styles.input} 
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Winter Hackathon 2026"
            />
          </div>
          <div className={styles.formGroup}>
            <label className={styles.label}>URL Slug *</label>
            <input 
              className={styles.input} 
              value={slug}
              onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '-'))}
              placeholder="e.g. winter-hackathon-2026"
            />
          </div>
          <div className={styles.formGroup}>
            <label className={styles.label}>Max Team Size</label>
            <input 
              type="number"
              min="1"
              max="20"
              className={styles.input} 
              value={maxTeamSize}
              onChange={(e) => setMaxTeamSize(Number(e.target.value))}
            />
          </div>
        </div>
      )}

      {step === 2 && (
        <div>
          <div className={styles.formGroup}>
            <label className={styles.label}>Description</label>
            <textarea 
              className={styles.textarea} 
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe your event rules, themes, and prizes..."
            />
          </div>
        </div>
      )}

      {step === 3 && (
        <div style={{ textAlign: 'center' }}>
          <h3>Ready to Launch?</h3>
          <p style={{ color: 'var(--text-secondary)', marginTop: '1rem', marginBottom: '2rem' }}>
            You will be able to edit these details and manage the timeline from the organizer dashboard.
          </p>
          <div style={{ padding: '1rem', background: 'var(--bg-darker)', borderRadius: '8px', textAlign: 'left' }}>
            <p><strong>Title:</strong> {title}</p>
            <p><strong>URL:</strong> /events/{slug}</p>
            <p><strong>Max Team Size:</strong> {maxTeamSize}</p>
          </div>
        </div>
      )}

      <div className={styles.actions}>
        <button 
          className={styles.buttonSecondary} 
          onClick={handleBack}
          disabled={step === 1 || loading}
        >
          Back
        </button>
        
        {step < 3 ? (
          <button 
            className={styles.buttonPrimary} 
            onClick={handleNext}
            disabled={step === 1 && (!title || !slug)}
          >
            Next
          </button>
        ) : (
          <button 
            className={styles.buttonPrimary} 
            onClick={handleSubmit}
            disabled={loading}
          >
            {loading ? 'Creating...' : 'Create Event'}
          </button>
        )}
      </div>
    </div>
  );
}
