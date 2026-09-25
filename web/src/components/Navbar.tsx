'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useState, useRef, useEffect } from 'react';
import { ChevronDown } from 'lucide-react';
import { useAuth } from '@/context/AuthContext';
import styles from './Navbar.module.css';

const NAV_LINKS = [
  { href: '/events', label: 'Events' },
];

const AUTH_NAV_LINKS = [
  { href: '/events', label: 'Events' },
  { href: '/team', label: 'My Team' },
  { href: '/submission', label: 'My Submission' },
];

export default function Navbar() {
  const { user, isAuthenticated, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  async function handleLogout() {
    setDropdownOpen(false);
    await logout();
    router.push('/login');
  }

  const links = isAuthenticated ? AUTH_NAV_LINKS : NAV_LINKS;

  return (
    <nav className={styles.nav} aria-label="Main navigation">
      <div className={styles.inner}>
        <Link href="/" className={styles.wordmark}>
          Dogfood
        </Link>

        <ul className={styles.links} role="list">
          {links.map(({ href, label }) => (
            <li key={href}>
              <Link
                href={href}
                className={`${styles.navLink} ${pathname === href ? styles.active : ''}`}
              >
                {label}
              </Link>
            </li>
          ))}
          {isAuthenticated && user?.isAdmin && (
            <li>
              <Link
                href="/admin"
                className={`${styles.navLink} ${pathname.startsWith('/admin') ? styles.active : ''}`}
              >
                Admin
              </Link>
            </li>
          )}
        </ul>

        <div className={styles.authArea}>
          {isAuthenticated && user ? (
            <div className={styles.dropdown} ref={dropdownRef}>
              <button
                id="user-menu-btn"
                className={styles.userBtn}
                onClick={() => setDropdownOpen((v) => !v)}
                aria-expanded={dropdownOpen}
                aria-haspopup="menu"
              >
                <span className={styles.avatar} aria-hidden="true">
                  {user.displayName.charAt(0).toUpperCase()}
                </span>
                <span className={styles.displayName}>{user.displayName}</span>
                <ChevronDown size={14} />
              </button>

              {dropdownOpen && (
                <div className={styles.dropdownMenu} role="menu">
                  <div className={styles.dropdownEmail}>{user.email}</div>
                  <button
                    id="logout-btn"
                    className={styles.dropdownItem}
                    role="menuitem"
                    onClick={handleLogout}
                  >
                    Sign out
                  </button>
                </div>
              )}
            </div>
          ) : (
            <div className={styles.authLinks}>
              <Link href="/login" className={styles.navLink}>
                Sign in
              </Link>
              <Link href="/register" className="btn" style={{ padding: '0.4rem 0.875rem', fontSize: '0.875rem' }}>
                Register
              </Link>
            </div>
          )}
        </div>
      </div>
    </nav>
  );
}
