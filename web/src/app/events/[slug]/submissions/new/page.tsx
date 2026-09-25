'use client';

import { useState, type FormEvent } from 'react';
import { useRouter, useParams } from 'next/navigation';
import Link from 'next/link';
import { api, ApiClientError } from '@/lib/api';
import type { CreateSubmissionRequest, SubmissionDTO } from '@/types/api';
import styles from './new.module.css';

export default function NewSubmissionPage() {
  const router = useRouter();
  const params = useParams();
  const slug = params.slug as string;

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [repoUrl, setRepoUrl] = useState('');
  const [demoUrl, setDemoUrl] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!title) return;

    setError(null);
    setIsSubmitting(true);

    try {
      const payload: CreateSubmissionRequest = {
        title,
        ...(description && { description }),
        ...(repoUrl && { repoUrl }),
        ...(demoUrl && { demoUrl }),
      };

      const res = await api.post<SubmissionDTO>(`/events/${slug}/submissions`, payload);
      
      // Navigate to the submission details page or event dashboard
      router.push(`/events/${slug}/submissions/${res.data.submissionId}`);
    } catch (err) {
      if (err instanceof ApiClientError) {
        if (err.code === 'DUPLICATE_RESOURCE') {
          setError('Your team already has a submission for this event.');
        } else if (err.code === 'INVARIANT_VIOLATION') {
          setError('Submissions are not open for this event.');
        } else if (err.code === 'FORBIDDEN') {
          setError('You must be a team member to create a submission.');
        } else {
          setError(err.message);
        }
      } else {
        setError('Something went wrong. Try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className={styles.page}>
      <main className={styles.form}>
        <header className={styles.header}>
          <h1 className={styles.title}>Create Submission</h1>
          <p className={styles.subtitle}>Draft your project submission for the hackathon.</p>
        </header>

        <form onSubmit={handleSubmit} noValidate>
          <div className={styles.field}>
            <label htmlFor="title" className={styles.label}>
              Project Title *
            </label>
            <input
              id="title"
              type="text"
              className="input"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Enter your project title"
              required
              disabled={isSubmitting}
            />
          </div>

          <div className={styles.field}>
            <label htmlFor="description" className={styles.label}>
              Description
            </label>
            <textarea
              id="description"
              className={`input ${styles.textarea}`}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What does your project do?"
              disabled={isSubmitting}
            />
          </div>

          <div className={styles.field}>
            <label htmlFor="repoUrl" className={styles.label}>
              Repository URL
            </label>
            <input
              id="repoUrl"
              type="url"
              className="input"
              value={repoUrl}
              onChange={(e) => setRepoUrl(e.target.value)}
              placeholder="https://github.com/your-username/repo"
              disabled={isSubmitting}
            />
          </div>

          <div className={styles.field}>
            <label htmlFor="demoUrl" className={styles.label}>
              Demo URL
            </label>
            <input
              id="demoUrl"
              type="url"
              className="input"
              value={demoUrl}
              onChange={(e) => setDemoUrl(e.target.value)}
              placeholder="https://your-demo-url.com"
              disabled={isSubmitting}
            />
          </div>

          {error && (
            <p className={`error-text ${styles.errorMsg}`} role="alert">
              {error}
            </p>
          )}

          <button
            id="create-submission-submit"
            type="submit"
            className={`btn ${styles.submitBtn}`}
            disabled={isSubmitting || !title}
          >
            {isSubmitting && <span className="spinner" aria-hidden="true" />}
            {isSubmitting ? 'Creating draft…' : 'Create Draft'}
          </button>
        </form>

        <p className={styles.footer}>
          <Link href={`/events/${slug}`}>Back to Event</Link>
        </p>
      </main>
    </div>
  );
}
