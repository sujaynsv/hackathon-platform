'use client';

import { useState, useEffect, type FormEvent } from 'react';
import { useRouter, useParams } from 'next/navigation';
import Link from 'next/link';
import { api, ApiClientError } from '@/lib/api';
import type { SubmissionDetail, UpdateSubmissionRequest, FileDTO } from '@/types/api';
import styles from './submission.module.css';

export default function SubmissionDashboardPage() {
  const router = useRouter();
  const params = useParams();
  const slug = params.slug as string;
  const id = params.id as string;

  const [submission, setSubmission] = useState<SubmissionDetail | null>(null);
  const [files, setFiles] = useState<FileDTO[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  // Form state
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [repoUrl, setRepoUrl] = useState('');
  const [demoUrl, setDemoUrl] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    fetchSubmission();
  }, [id]);

  async function fetchSubmission() {
    try {
      setLoading(true);
      const res = await api.get<SubmissionDetail>(`/submissions/${id}`);
      setSubmission(res.data);
      setTitle(res.data.title || '');
      setDescription(res.data.description || '');
      setRepoUrl(res.data.repoUrl || '');
      setDemoUrl(res.data.demoUrl || '');

      const filesRes = await api.get<FileDTO[]>(`/submissions/${id}/files`);
      setFiles(filesRes.data || []);
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('Failed to load submission.');
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleSaveDraft(e: FormEvent) {
    e.preventDefault();
    if (!title) return;

    setError(null);
    setIsSaving(true);

    try {
      const payload: UpdateSubmissionRequest = {
        title,
        description,
        repoUrl,
        demoUrl,
      };

      await api.patch(`/submissions/${id}`, payload);
      await fetchSubmission(); // Refresh
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('Failed to save draft.');
      }
    } finally {
      setIsSaving(false);
    }
  }

  async function handleUploadFile(e: React.ChangeEvent<HTMLInputElement>) {
    if (!e.target.files || e.target.files.length === 0) return;
    const file = e.target.files[0];

    // Basic client-side validation
    if (file.size > 10 * 1024 * 1024) {
      setError('File must be less than 10MB');
      return;
    }

    setIsUploading(true);
    setError(null);

    try {
      const formData = new FormData();
      formData.append('file', file);
      // Auto-assign role based on file type for now
      formData.append('role', file.type.startsWith('image/') ? 'cover' : 'attachment');

      await api.post(`/submissions/${id}/upload`, formData);
      await fetchSubmission(); // Refresh files
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('Failed to upload file.');
      }
    } finally {
      setIsUploading(false);
      // Reset input
      e.target.value = '';
    }
  }

  async function handleFinalSubmit() {
    if (!confirm('Are you sure you want to submit? You cannot edit this after submission.')) {
      return;
    }

    setError(null);
    setIsSubmitting(true);

    try {
      await api.post(`/submissions/${id}/submit`, {});
      await fetchSubmission(); // Refresh to see updated status
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('Failed to submit.');
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  if (loading) {
    return <div className={styles.page}>Loading submission...</div>;
  }

  if (error && !submission) {
    return (
      <div className={styles.page}>
        <div className="error-text">{error}</div>
        <Link href={`/events/${slug}`}>Back to event</Link>
      </div>
    );
  }

  if (!submission) return null;

  const isSubmitted = submission.status === 'submitted';

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.title}>Submission Dashboard</h1>
        <p className={styles.subtitle}>Edit your draft, upload materials, and finalize your submission.</p>
        <div className={`${styles.statusBadge} ${isSubmitted ? styles.submitted : ''}`}>
          {submission.status.toUpperCase()}
        </div>
      </header>

      {error && (
        <p className={`error-text`} style={{ marginBottom: '1rem' }} role="alert">
          {error}
        </p>
      )}

      <div className={styles.form}>
        <form onSubmit={handleSaveDraft}>
          <h2 className={styles.sectionTitle}>Project Details</h2>
          
          <div className={styles.field} style={{ marginBottom: '1rem' }}>
            <label htmlFor="title" className={styles.label}>Project Title *</label>
            <input
              id="title"
              type="text"
              className="input"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={isSubmitted || isSaving}
              required
            />
          </div>

          <div className={styles.field} style={{ marginBottom: '1rem' }}>
            <label htmlFor="description" className={styles.label}>Description</label>
            <textarea
              id="description"
              className={`input ${styles.textarea}`}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={isSubmitted || isSaving}
            />
          </div>

          <div className={styles.field} style={{ marginBottom: '1rem' }}>
            <label htmlFor="repoUrl" className={styles.label}>Repository URL</label>
            <input
              id="repoUrl"
              type="url"
              className="input"
              value={repoUrl}
              onChange={(e) => setRepoUrl(e.target.value)}
              disabled={isSubmitted || isSaving}
            />
          </div>

          <div className={styles.field} style={{ marginBottom: '1rem' }}>
            <label htmlFor="demoUrl" className={styles.label}>Demo URL</label>
            <input
              id="demoUrl"
              type="url"
              className="input"
              value={demoUrl}
              onChange={(e) => setDemoUrl(e.target.value)}
              disabled={isSubmitted || isSaving}
            />
          </div>

          {!isSubmitted && (
            <button
              type="submit"
              className="btn"
              disabled={isSaving || !title}
            >
              {isSaving ? 'Saving...' : 'Save Draft'}
            </button>
          )}
        </form>

        <div style={{ marginTop: '2rem' }}>
          <h2 className={styles.sectionTitle}>Attachments</h2>
          
          {!isSubmitted && (
            <div className={styles.uploadSection}>
              <label htmlFor="fileUpload" className={styles.uploadLabel}>
                {isUploading ? 'Uploading...' : 'Click to select a file to upload (Max 10MB)'}
                <input
                  id="fileUpload"
                  type="file"
                  className={styles.uploadInput}
                  onChange={handleUploadFile}
                  disabled={isUploading}
                  accept="image/png,image/jpeg,application/pdf"
                />
              </label>
            </div>
          )}

          {files.length > 0 ? (
            <div className={styles.fileList}>
              {files.map((file) => (
                <div key={file.id} className={styles.fileItem}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
                    <span className={styles.fileName}>{file.name}</span>
                    <span className={styles.fileRole}>{file.role}</span>
                  </div>
                  <a href={file.url} target="_blank" rel="noopener noreferrer" style={{ fontSize: '0.875rem' }}>View</a>
                </div>
              ))}
            </div>
          ) : (
            <p style={{ color: 'var(--text-secondary)' }}>No files uploaded yet.</p>
          )}
        </div>

        <div className={styles.actions}>
          <Link href={`/events/${slug}`}>Back to Event</Link>
          
          {!isSubmitted && (
            <button
              className={`btn ${styles.submitBtn}`}
              onClick={handleFinalSubmit}
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Submitting...' : 'Finalize Submission'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
