import React from 'react';
import type { EventDetailDTO } from '@/types/api';

interface DateTimelineProps {
  event: EventDetailDTO;
}

export function DateTimeline({ event }: DateTimelineProps) {
  const milestones = [
    { label: 'Registration Opens', date: event.registrationOpensAt },
    { label: 'Registration Closes', date: event.registrationClosesAt },
    { label: 'Submissions Due', date: event.submissionDeadlineAt },
    { label: 'Judging Ends', date: event.judgingDeadlineAt },
    { label: 'Voting Opens', date: event.votingOpensAt },
    { label: 'Voting Closes', date: event.votingClosesAt },
  ].filter(m => m.date);

  const now = new Date();

  return (
    <div className="timeline-container">
      <h3 className="timeline-title">Event Timeline</h3>
      <div className="timeline">
        {milestones.length === 0 ? (
          <p className="empty-text">No dates scheduled yet.</p>
        ) : (
          milestones.map((m, i) => {
            const dateObj = new Date(m.date!);
            const isPast = dateObj < now;
            
            return (
              <div key={i} className={`timeline-item ${isPast ? 'past' : 'future'}`}>
                <div className="marker"></div>
                <div className="content">
                  <div className="label">{m.label}</div>
                  <div className="date">
                    {new Intl.DateTimeFormat('default', {
                      month: 'short',
                      day: 'numeric',
                      hour: 'numeric',
                      minute: 'numeric',
                    }).format(dateObj)}
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>

      <style jsx>{`
        .timeline-container {
          background: var(--surface);
          border: 1px solid var(--border);
          border-radius: 12px;
          padding: 1.5rem;
        }
        
        .timeline-title {
          font-size: 1.1rem;
          font-weight: 600;
          margin-bottom: 1.5rem;
          color: var(--text-primary);
        }
        
        .timeline {
          display: flex;
          flex-direction: column;
          gap: 1.5rem;
          position: relative;
        }
        
        .timeline::before {
          content: '';
          position: absolute;
          left: 6px;
          top: 0;
          bottom: 0;
          width: 2px;
          background: var(--border);
        }
        
        .empty-text {
          color: var(--text-secondary);
          font-style: italic;
        }
        
        .timeline-item {
          display: flex;
          gap: 1rem;
          position: relative;
        }
        
        .marker {
          width: 14px;
          height: 14px;
          border-radius: 50%;
          background: var(--bg-darker);
          border: 2px solid var(--border);
          position: relative;
          z-index: 1;
        }
        
        .timeline-item.past .marker {
          background: var(--primary);
          border-color: var(--primary);
        }
        
        .content {
          margin-top: -3px;
        }
        
        .label {
          font-weight: 500;
          color: var(--text-primary);
          margin-bottom: 0.25rem;
        }
        
        .timeline-item.past .label {
          color: var(--text-secondary);
        }
        
        .date {
          font-size: 0.85rem;
          color: var(--text-secondary);
        }
      `}</style>
    </div>
  );
}
