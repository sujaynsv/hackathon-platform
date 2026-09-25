'use client';

import { useState, type FormEvent } from 'react';
import Link from 'next/link';
import { Eye, EyeOff } from 'lucide-react';
import { useAuth, ApiClientError } from '@/context/AuthContext';
import styles from './register.module.css';

function passwordStrength(pw: string): 'weak' | 'fair' | 'strong' {
  if (pw.length < 8) return 'weak';
  const hasUpper = /[A-Z]/.test(pw);
  const hasLower = /[a-z]/.test(pw);
  const hasDigit = /\d/.test(pw);
  const hasSymbol = /[^a-zA-Z0-9]/.test(pw);
  const score = [hasUpper, hasLower, hasDigit, hasSymbol].filter(Boolean).length;
  if (score >= 4 && pw.length >= 12) return 'strong';
  if (score >= 3) return 'fair';
  return 'weak';
}

export default function RegisterPage() {
  const { register } = useAuth();

  const [displayName, setDisplayName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [verifyPending, setVerifyPending] = useState(false);

  const strength = password ? passwordStrength(password) : null;
  const mismatch = confirm && password !== confirm;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!displayName || !email || !password || !confirm) return;
    if (password !== confirm) {
      setError('Passwords do not match.');
      return;
    }

    setError(null);
    setIsSubmitting(true);

    try {
      await register({ displayName, email, password });
      // If registration requires verification, show message instead of redirecting
      setVerifyPending(true);
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError('Something went wrong. Try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  if (verifyPending) {
    return (
      <div className={styles.page}>
        <div className={styles.form}>
          <p className={styles.verifyMsg}>
            Account created. Check your email for a verification link before signing in.
          </p>
          <p className={styles.footer}>
            <Link href="/login">Back to sign in</Link>
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <main className={styles.form}>
        <header className={styles.header}>
          <span className={styles.wordmark}>Dogfood</span>
          <h1 className={styles.title}>Create account</h1>
        </header>

        <form onSubmit={handleSubmit} noValidate>
          <div className={styles.field}>
            <label htmlFor="displayName" className={styles.label}>
              Display name
            </label>
            <input
              id="displayName"
              type="text"
              className="input"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              autoComplete="name"
              required
              disabled={isSubmitting}
            />
          </div>

          <div className={styles.field}>
            <label htmlFor="email" className={styles.label}>
              Email
            </label>
            <input
              id="email"
              type="email"
              className="input"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              required
              disabled={isSubmitting}
            />
          </div>

          <div className={styles.field}>
            <label htmlFor="password" className={styles.label}>
              Password
            </label>
            <div className={styles.passwordWrapper}>
              <input
                id="password"
                type={showPassword ? 'text' : 'password'}
                className="input"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="new-password"
                required
                disabled={isSubmitting}
              />
              <button
                type="button"
                className={styles.eyeBtn}
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
            {strength && (
              <div className={styles.strengthRow} aria-label={`Password strength: ${strength}`}>
                <div className={`${styles.strengthBar} ${styles[strength]}`} />
                <span className={`${styles.strengthLabel} ${styles[strength]}`}>
                  {strength === 'weak' && 'Weak'}
                  {strength === 'fair' && 'Fair'}
                  {strength === 'strong' && 'Strong'}
                </span>
              </div>
            )}
          </div>

          <div className={styles.field}>
            <label htmlFor="confirm" className={styles.label}>
              Confirm password
            </label>
            <input
              id="confirm"
              type={showPassword ? 'text' : 'password'}
              className={`input${mismatch ? ' input-error' : ''}`}
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              autoComplete="new-password"
              required
              disabled={isSubmitting}
            />
            {mismatch && (
              <p className="error-text" style={{ marginTop: '0.375rem' }}>
                Passwords do not match.
              </p>
            )}
          </div>

          {error && (
            <p className={`error-text ${styles.errorMsg}`} role="alert">
              {error}
            </p>
          )}

          <button
            id="register-submit"
            type="submit"
            className={`btn ${styles.submitBtn}`}
            disabled={isSubmitting || !displayName || !email || !password || !confirm || !!mismatch}
          >
            {isSubmitting && <span className="spinner" aria-hidden="true" />}
            {isSubmitting ? 'Creating account…' : 'Create account'}
          </button>
        </form>

        <p className={styles.footer}>
          Already have an account?{' '}
          <Link href="/login">Sign in</Link>
        </p>
      </main>
    </div>
  );
}
