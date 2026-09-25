import Link from 'next/link';
import type { Metadata } from 'next';
import { Calendar, Users, CheckSquare, BarChart3, ArrowRight, ShieldCheck, Cpu } from 'lucide-react';
import styles from './home.module.css';

export const metadata: Metadata = {
  title: 'Dogfood — Hackathon Management Platform',
  description:
    'The developer-first platform for hosting, hacking, and judging high-velocity hackathons with offline-capable architecture.',
};

export default function HomePage() {
  return (
    <div className={styles.page}>
      <main className={styles.main}>
        {/* Hero Section */}
        <section className={styles.hero}>
          <div className={styles.eyebrow}>
            <Cpu size={14} />
            <span>Developer-First Infrastructure</span>
          </div>
          <h1 className={styles.title}>
            Dogfood Hackathon Platform
          </h1>
          <p className={styles.description}>
            A resilient platform for hosting and scoring high-velocity hackathons.
            Built with strict state machines, role-based access, rubric-normalized scoring,
            and an offline-capable architecture.
          </p>
          <div className={styles.actions}>
            <Link href="/register" className={`btn ${styles.btnLg}`}>
              Create account
              <ArrowRight size={16} />
            </Link>
            <Link href="/login" className={`btn btn-secondary ${styles.btnLg}`}>
              Sign in
            </Link>
            <Link href="/events" className={`btn btn-secondary ${styles.btnLg}`}>
              Browse events
            </Link>
          </div>
        </section>

        {/* Core Capabilities */}
        <section className={styles.section}>
          <div className={styles.sectionHeader}>
            <p className={styles.sectionLabel}>Core Architecture</p>
            <h2 className={styles.sectionTitle}>Engineered for integrity and scale</h2>
          </div>

          <div className={styles.grid}>
            <div className={styles.feature}>
              <div className={styles.featureHeader}>
                <div className={styles.featureIcon}>
                  <Calendar size={18} />
                </div>
                <div>
                  <span className={styles.featureNumber}>01</span>
                  <h3 className={styles.featureName}>Event Orchestration</h3>
                </div>
              </div>
              <p className={styles.featureDesc}>
                Strict lifecycle state transitions from draft to registration, hacking,
                submission deadlines, judging windows, and finalized winners.
              </p>
            </div>

            <div className={styles.feature}>
              <div className={styles.featureHeader}>
                <div className={styles.featureIcon}>
                  <Users size={18} />
                </div>
                <div>
                  <span className={styles.featureNumber}>02</span>
                  <h3 className={styles.featureName}>Team Assembly</h3>
                </div>
              </div>
              <p className={styles.featureDesc}>
                Role-based member invitation flows, captain delegation, and roster locking
                guaranteed by transactional concurrency checks.
              </p>
            </div>

            <div className={styles.feature}>
              <div className={styles.featureHeader}>
                <div className={styles.featureIcon}>
                  <CheckSquare size={18} />
                </div>
                <div>
                  <span className={styles.featureNumber}>03</span>
                  <h3 className={styles.featureName}>Rubric-Based Judging</h3>
                </div>
              </div>
              <p className={styles.featureDesc}>
                Multi-criteria rubrics with automated z-score normalization across judges to
                eliminate individual scoring bias.
              </p>
            </div>

            <div className={styles.feature}>
              <div className={styles.featureHeader}>
                <div className={styles.featureIcon}>
                  <BarChart3 size={18} />
                </div>
                <div>
                  <span className={styles.featureNumber}>04</span>
                  <h3 className={styles.featureName}>Public Voting & Auditing</h3>
                </div>
              </div>
              <p className={styles.featureDesc}>
                Fraud-resistant public voting with IP salt hashing, token bucket rate-limiting,
                and database-enforced append-only audit logs.
              </p>
            </div>
          </div>
        </section>

        {/* Technical Foundation */}
        <section className={styles.section}>
          <div className={styles.sectionHeader}>
            <p className={styles.sectionLabel}>Technical Stack</p>
            <h2 className={styles.sectionTitle}>Modern, modular foundation</h2>
          </div>

          <div className={styles.specTableWrapper}>
            <table className={styles.specTable}>
              <thead>
                <tr>
                  <th>Layer</th>
                  <th>Technology</th>
                  <th>Design Pattern / Role</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>Backend API</td>
                  <td>Go 1.23 / Chi v5</td>
                  <td>Modular Monolith + Hexagonal Architecture</td>
                </tr>
                <tr>
                  <td>Frontend App</td>
                  <td>Next.js 14 / TypeScript</td>
                  <td>React Server Components + CSS Modules</td>
                </tr>
                <tr>
                  <td>Database</td>
                  <td>PostgreSQL 16 (pgx/sqlx)</td>
                  <td>Strict relational integrity, 19 normalized tables</td>
                </tr>
                <tr>
                  <td>Caching & Limiting</td>
                  <td>Redis 7</td>
                  <td>Token bucket rate limiting & blacklist caching</td>
                </tr>
                <tr>
                  <td>Object Storage</td>
                  <td>MinIO (S3 compatible)</td>
                  <td>Direct-to-storage artifact and project uploads</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        {/* Final CTA */}
        <section className={styles.ctaBox}>
          <div>
            <h2 className={styles.ctaTitle}>Ready to get started?</h2>
            <p className={styles.ctaSub}>
              Create your account to join an active event or register your team.
            </p>
          </div>
          <div className={styles.ctaActions}>
            <Link href="/register" className="btn">
              Create account
            </Link>
            <Link href="/login" className="btn btn-secondary">
              Sign in
            </Link>
          </div>
        </section>
      </main>

      <footer className={styles.footer}>
        <div className={styles.footerInner}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            <ShieldCheck size={16} style={{ color: 'var(--accent)' }} />
            <span>Dogfood Platform v1.0.0 &middot; Modular Monolith</span>
          </div>
          <div className={styles.footerLinks}>
            <Link href="/login">Sign in</Link>
            <Link href="/register">Register</Link>
            <Link href="/verify-email">Verify Email</Link>
            <Link href="/events">Events</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
