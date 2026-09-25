'use client';

import { useState, useEffect, type FormEvent } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { CheckCircle2, AlertCircle, ArrowRight, RefreshCw } from 'lucide-react';
import { apiClient, ApiClientError } from '@/lib/api';
import styles from './verify-email.module.css';

export default function VerifyEmailClient() {
  const searchParams = useSearchParams();
  const tokenParam = searchParams.get('token');

  const [tokenInput, setTokenInput] = useState('');
  const [status, setStatus] = useState<'idle' | 'verifying' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  async function executeVerification(token: string) {
    if (!token.trim()) return;
    setStatus('verifying');
    setErrorMessage(null);

    try {
      await apiClient.verifyEmail(token.trim());
      setStatus('success');
    } catch (err) {
      setStatus('error');
      if (err instanceof ApiClientError) {
        if (err.code === 'INVALID_STATE_TRANSITION') {
          setErrorMessage('This verification link has already been used.');
        } else if (err.code === 'DEADLINE_PASSED') {
          setErrorMessage('This verification link has expired. Please register again.');
        } else if (err.code === 'NOT_FOUND') {
          setErrorMessage('Invalid verification token. Please check the link or code.');
        } else {
          setErrorMessage(err.message || 'Verification failed. Please try again.');
        }
      } else {
        setErrorMessage('Unable to connect to the verification service. Please try again.');
      }
    }
  }

  useEffect(() => {
    if (tokenParam) {
      setTokenInput(tokenParam);
      executeVerification(tokenParam);
    }
  }, [tokenParam]);

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    executeVerification(tokenInput);
  }

  function handleReset() {
    setStatus('idle');
    setErrorMessage(null);
  }

  return (
    <div className={styles.page}>
      <main className={styles.card}>
        <header className={styles.header}>
          <span className={styles.wordmark}>Dogfood</span>
          <h1 className={styles.title}>Email verification</h1>
          {status === 'idle' && (
            <p className={styles.subtitle}>
              Confirm your email address to activate your account and access hackathons.
            </p>
          )}
        </header>

        {status === 'verifying' && (
          <div className={styles.statusBox}>
            <div className={styles.statusIcon} style={{ background: 'rgba(212, 160, 85, 0.1)', color: 'var(--accent)' }}>
              <RefreshCw size={22} className="spinner" />
            </div>
            <h2 className={styles.statusTitle}>Verifying your email</h2>
            <p className={styles.statusText}>Checking your verification token with the server...</p>
          </div>
        )}

        {status === 'success' && (
          <div className={styles.statusBox}>
            <div className={`${styles.statusIcon} ${styles.statusIconSuccess}`}>
              <CheckCircle2 size={24} />
            </div>
            <h2 className={styles.statusTitle}>Email verified</h2>
            <p className={styles.statusText}>
              Your email has been confirmed. You can now sign in to your account.
            </p>
            <Link href="/login" className={`btn ${styles.actionBtn}`}>
              Continue to sign in
              <ArrowRight size={16} style={{ marginLeft: '0.5rem' }} />
            </Link>
          </div>
        )}

        {status === 'error' && (
          <div className={styles.statusBox}>
            <div className={`${styles.statusIcon} ${styles.statusIconError}`}>
              <AlertCircle size={24} />
            </div>
            <h2 className={styles.statusTitle}>Verification failed</h2>
            <p className={styles.statusText}>{errorMessage}</p>
            <button type="button" onClick={handleReset} className={`btn btn-secondary ${styles.actionBtn}`}>
              Enter token manually
            </button>
          </div>
        )}

        {status === 'idle' && (
          <>
            <form onSubmit={handleSubmit} noValidate>
              <div className={styles.field}>
                <label htmlFor="token" className={styles.label}>
                  Verification token
                </label>
                <input
                  id="token"
                  type="text"
                  className="input"
                  value={tokenInput}
                  onChange={(e) => setTokenInput(e.target.value)}
                  placeholder="Paste your verification token here"
                  required
                />
              </div>

              <button
                type="submit"
                className={`btn ${styles.submitBtn}`}
                disabled={!tokenInput.trim()}
              >
                Verify email
              </button>
            </form>


          </>
        )}

        <p className={styles.footer}>
          Return to <Link href="/login">Sign in</Link> or <Link href="/register">Create account</Link>
        </p>
      </main>
    </div>
  );
}
