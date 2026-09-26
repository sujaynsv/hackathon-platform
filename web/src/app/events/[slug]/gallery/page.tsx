'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { api, ApiClientError } from '@/lib/api';
import type { SubmissionGalleryRow } from '@/types/api';
import styles from './gallery.module.css';

export default function GalleryPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const router = useRouter();
  
  const slug = params.slug as string;
  const pageParam = searchParams.get('page') || '1';
  const trackParam = searchParams.get('trackId') || '';
  const sortParam = searchParams.get('sort') || 'newest';

  const [submissions, setSubmissions] = useState<SubmissionGalleryRow[]>([]);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchGallery();
  }, [slug, pageParam, trackParam, sortParam]);

  async function fetchGallery() {
    setLoading(true);
    setError(null);
    try {
      const query = new URLSearchParams({
        page: pageParam,
        size: '12',
        sort: sortParam,
      });
      if (trackParam) {
        query.set('trackId', trackParam);
      }

      const res = await api.get<SubmissionGalleryRow[]>(`/events/${slug}/submissions?${query.toString()}`);
      
      // Handle the unwrapped list format
      setSubmissions(res.data);
      setTotalPages(res.meta.totalPages || 1);
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('Failed to load gallery.');
      }
    } finally {
      setLoading(false);
    }
  }

  function handleFilterChange(key: string, value: string) {
    const newParams = new URLSearchParams(searchParams.toString());
    if (value) {
      newParams.set(key, value);
    } else {
      newParams.delete(key);
    }
    newParams.set('page', '1'); // reset to page 1
    router.push(`/events/${slug}/gallery?${newParams.toString()}`);
  }

  function handlePageChange(newPage: number) {
    if (newPage < 1 || newPage > totalPages) return;
    const newParams = new URLSearchParams(searchParams.toString());
    newParams.set('page', newPage.toString());
    router.push(`/events/${slug}/gallery?${newParams.toString()}`);
  }

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.title}>Project Gallery</h1>
        <p className={styles.subtitle}>Explore the amazing projects built for this event.</p>
      </header>

      <div className={styles.controls}>
        <div className={styles.filterGroup}>
          <select 
            className={styles.select}
            value={trackParam}
            onChange={(e) => handleFilterChange('trackId', e.target.value)}
          >
            <option value="">All Tracks</option>
            {/* Tracks should ideally be fetched from the event details, but keeping it simple for now */}
            <option value="software">Software Track</option>
            <option value="hardware">Hardware Track</option>
          </select>
        </div>

        <div className={styles.filterGroup}>
          <select
            className={styles.select}
            value={sortParam}
            onChange={(e) => handleFilterChange('sort', e.target.value)}
          >
            <option value="newest">Newest First</option>
            <option value="oldest">Oldest First</option>
            <option value="score">Highest Score</option>
          </select>
        </div>
      </div>

      {error && <div className="error-text" style={{ marginBottom: '2rem', textAlign: 'center' }}>{error}</div>}

      {loading ? (
        <div style={{ textAlign: 'center', padding: '4rem 0', color: 'var(--text-secondary)' }}>Loading projects...</div>
      ) : (
        <>
          {submissions.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '4rem 0', color: 'var(--text-secondary)' }}>
              No submissions found.
            </div>
          ) : (
            <div className={styles.grid}>
              {submissions.map((sub) => (
                <div key={sub.SubmissionID} className={styles.card}>
                  <div 
                    className={styles.cover} 
                    style={{ backgroundImage: sub.CoverURL ? `url(${sub.CoverURL})` : 'none' }}
                  />
                  <div className={styles.content}>
                    <h3 className={styles.submissionTitle}>{sub.Title}</h3>
                    <p className={styles.teamName}>by Team {sub.TeamName}</p>
                    
                    <div className={styles.badges}>
                      {sub.TrackName && <span className={styles.badge}>{sub.TrackName}</span>}
                      {sub.Status === 'winner' && <span className={styles.badge} style={{ background: 'var(--success)', color: 'white', borderColor: 'var(--success)' }}>Winner</span>}
                    </div>

                    <div className={styles.footer}>
                      <div className={styles.links}>
                        {sub.RepoURL && (
                          <a href={sub.RepoURL} target="_blank" rel="noopener noreferrer" className={styles.link}>
                            Repository
                          </a>
                        )}
                        {sub.DemoURL && (
                          <a href={sub.DemoURL} target="_blank" rel="noopener noreferrer" className={styles.link}>
                            Live Demo
                          </a>
                        )}
                      </div>
                      
                      {sub.FinalScore !== null && (
                        <div className={styles.score}>{sub.FinalScore.toFixed(1)} / 100</div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {totalPages > 1 && (
            <div className={styles.pagination}>
              <button 
                className="btn" 
                onClick={() => handlePageChange(parseInt(pageParam) - 1)}
                disabled={parseInt(pageParam) <= 1}
              >
                Previous
              </button>
              <span className={styles.pageInfo}>
                Page {pageParam} of {totalPages}
              </span>
              <button 
                className="btn" 
                onClick={() => handlePageChange(parseInt(pageParam) + 1)}
                disabled={parseInt(pageParam) >= totalPages}
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
