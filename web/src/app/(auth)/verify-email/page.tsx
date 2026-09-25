import { Suspense } from 'react';
import type { Metadata } from 'next';
import VerifyEmailClient from './VerifyEmailClient';
import styles from './verify-email.module.css';

export const metadata: Metadata = {
  title: 'Verify Email — Dogfood Hackathon Platform',
  description: 'Verify your email address to activate your Dogfood account.',
};

export default function VerifyEmailPage() {
  return (
    <Suspense
      fallback={
        <div className={styles.page}>
          <div className={styles.card}>
            <div className={styles.statusBox}>
              <span className="spinner" aria-hidden="true" />
              <p className={styles.statusText} style={{ marginTop: '1rem', marginBottom: 0 }}>
                Loading verification...
              </p>
            </div>
          </div>
        </div>
      }
    >
      <VerifyEmailClient />
    </Suspense>
  );
}
