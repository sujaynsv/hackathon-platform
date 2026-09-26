import React from 'react';
import Link from 'next/link';
import { EventDTO } from '@/types/api';
import { StatusBadge } from './StatusBadge';

interface EventCardProps {
  event: EventDTO;
}

export function EventCard({ event }: EventCardProps) {
  const isRegistrationOpen = event.status === 'registration_open';
  
  return (
    <Link href={`/events/${event.slug}`} className="event-card">
      <div className="banner">
        {event.bannerUrl ? (
          <img src={event.bannerUrl} alt={event.title} className="banner-img" />
        ) : (
          <div className="banner-gradient" />
        )}
        <div className="banner-overlay" />
        <div className="status-container">
          <StatusBadge status={event.status} />
        </div>
      </div>
      
      <div className="content">
        <h3 className="title">{event.title}</h3>
        {event.description && (
          <p className="description">{event.description}</p>
        )}
        
        <div className="meta">
          {event.registrationClosesAt && isRegistrationOpen && (
            <div className="meta-item">
              <span className="icon">⏳</span>
              <span>Reg closes: {new Date(event.registrationClosesAt).toLocaleDateString()}</span>
            </div>
          )}
          <div className="meta-item">
            <span className="icon">👥</span>
            <span>Max Team: {event.maxTeamSize}</span>
          </div>
        </div>
      </div>

      <style jsx>{`
        .event-card {
          display: flex;
          flex-direction: column;
          background: var(--surface);
          border-radius: 12px;
          border: 1px solid var(--border);
          overflow: hidden;
          transition: all 0.3s ease;
          text-decoration: none;
          color: inherit;
          height: 100%;
        }
        
        .event-card:hover {
          transform: translateY(-4px);
          box-shadow: 0 12px 24px rgba(0,0,0,0.3);
          border-color: var(--primary);
        }
        
        .banner {
          position: relative;
          width: 100%;
          aspect-ratio: 16/9;
          background: var(--bg-darker);
          overflow: hidden;
        }
        
        .banner-img {
          width: 100%;
          height: 100%;
          object-fit: cover;
        }
        
        .banner-gradient {
          width: 100%;
          height: 100%;
          background: linear-gradient(135deg, #1f2937 0%, #0f172a 100%);
        }
        
        .banner-overlay {
          position: absolute;
          inset: 0;
          background: linear-gradient(to top, rgba(0,0,0,0.8) 0%, rgba(0,0,0,0) 100%);
        }
        
        .status-container {
          position: absolute;
          top: 1rem;
          right: 1rem;
        }
        
        .content {
          padding: 1.5rem;
          display: flex;
          flex-direction: column;
          flex-grow: 1;
        }
        
        .title {
          font-size: 1.25rem;
          font-weight: 700;
          margin-bottom: 0.5rem;
          color: var(--text-primary);
        }
        
        .description {
          font-size: 0.9rem;
          color: var(--text-secondary);
          margin-bottom: 1.5rem;
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
          flex-grow: 1;
        }
        
        .meta {
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
          margin-top: auto;
          font-size: 0.85rem;
          color: var(--text-secondary);
        }
        
        .meta-item {
          display: flex;
          align-items: center;
          gap: 0.5rem;
        }
      `}</style>
    </Link>
  );
}
