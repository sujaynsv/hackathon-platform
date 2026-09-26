import React from 'react';
import type { EventStatus } from '@/types/api';

const STATUS_CONFIG: Record<EventStatus, { label: string; colorClass: string }> = {
  draft: { label: 'Draft', colorClass: 'badge-gray' },
  registration_open: { label: 'Open for Registration', colorClass: 'badge-green' },
  submissions_open: { label: 'Accepting Submissions', colorClass: 'badge-blue' },
  judging: { label: 'Judging in Progress', colorClass: 'badge-purple' },
  voting: { label: 'Public Voting', colorClass: 'badge-yellow' },
  results_published: { label: 'Results Published', colorClass: 'badge-indigo' },
  archived: { label: 'Archived', colorClass: 'badge-gray' },
};

interface StatusBadgeProps {
  status: EventStatus;
  className?: string;
}

export function StatusBadge({ status, className = '' }: StatusBadgeProps) {
  const config = STATUS_CONFIG[status] || STATUS_CONFIG.draft;
  
  return (
    <span className={`status-badge ${config.colorClass} ${className}`}>
      {config.label}
      <style jsx>{`
        .status-badge {
          display: inline-block;
          padding: 0.25rem 0.75rem;
          border-radius: 9999px;
          font-size: 0.75rem;
          font-weight: 600;
          letter-spacing: 0.025em;
          text-transform: uppercase;
        }
        .badge-gray {
          background-color: rgba(156, 163, 175, 0.2);
          color: #9ca3af;
        }
        .badge-green {
          background-color: rgba(16, 185, 129, 0.2);
          color: #10b981;
        }
        .badge-blue {
          background-color: rgba(59, 130, 246, 0.2);
          color: #3b82f6;
        }
        .badge-purple {
          background-color: rgba(139, 92, 246, 0.2);
          color: #8b5cf6;
        }
        .badge-yellow {
          background-color: rgba(245, 158, 11, 0.2);
          color: #f59e0b;
        }
        .badge-indigo {
          background-color: rgba(99, 102, 241, 0.2);
          color: #6366f1;
        }
      `}</style>
    </span>
  );
}
