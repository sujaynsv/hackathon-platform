'use client';

import React, { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { apiClient } from '@/lib/api';
import { useMyTeam } from '@/hooks/useMyTeam';
import { TeamCard } from '@/components/teams/TeamCard';
import { CreateTeamForm } from '@/components/teams/CreateTeamForm';
import { JoinWithCodeForm } from '@/components/teams/JoinWithCodeForm';
import type { EventDetailDTO } from '@/types/api';
import Link from 'next/link';
import { useAuth } from '@/context/AuthContext';

export default function ParticipatePage() {
  const params = useParams();
  const router = useRouter();
  const slug = params.slug as string;
  const { user } = useAuth();

  const [event, setEvent] = useState<EventDetailDTO | null>(null);
  const [isLoadingEvent, setIsLoadingEvent] = useState(true);
  const [isRegistering, setIsRegistering] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { data: team, isLoading: isLoadingTeam, refetch: refetchTeam } = useMyTeam(slug);

  useEffect(() => {
    apiClient.getEvent(slug)
      .then(setEvent)
      .catch((err) => setError(err.message))
      .finally(() => setIsLoadingEvent(false));
  }, [slug]);

  const handleRegister = async () => {
    setIsRegistering(true);
    setError(null);
    try {
      await apiClient.registerForEvent(slug);
      // Reload event to get updated myRole
      const updatedEvent = await apiClient.getEvent(slug);
      setEvent(updatedEvent);
    } catch (err: any) {
      setError(err.message || 'Failed to register');
    } finally {
      setIsRegistering(false);
    }
  };

  const handleLeaveTeam = async () => {
    try {
      await apiClient.leaveTeam(slug);
      refetchTeam();
    } catch (err: any) {
      setError(err.message || 'Failed to leave team');
    }
  };

  if (isLoadingEvent) {
    return <div className="p-12 text-center text-gray-400">Loading...</div>;
  }

  if (!event) {
    return <div className="p-12 text-center text-red-400">Event not found.</div>;
  }

  const isRegistered = event.myRole === 'participant';
  const step = isRegistered ? (team ? 3 : 2) : 1;

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <div className="mb-8">
        <Link href={`/events/${slug}`} className="text-blue-400 hover:text-blue-300 text-sm flex items-center gap-2 mb-4">
          &larr; Back to {event.title}
        </Link>
        <h1 className="text-3xl font-bold text-white">Participate</h1>
      </div>

      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-400 rounded-lg p-4 mb-8">
          {error}
        </div>
      )}

      {/* Progress Steps */}
      <div className="flex items-center justify-between mb-12 relative">
        <div className="absolute left-0 top-1/2 -translate-y-1/2 w-full h-1 bg-gray-800 -z-10" />
        <div className={`absolute left-0 top-1/2 -translate-y-1/2 h-1 bg-blue-500 -z-10 transition-all duration-500 ${step === 1 ? 'w-0' : step === 2 ? 'w-1/2' : 'w-full'}`} />
        
        <div className="flex flex-col items-center gap-2">
          <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm ${step >= 1 ? 'bg-blue-500 text-white' : 'bg-gray-800 text-gray-500'}`}>1</div>
          <span className={`text-xs uppercase tracking-wider font-semibold ${step >= 1 ? 'text-blue-400' : 'text-gray-500'}`}>Register</span>
        </div>
        <div className="flex flex-col items-center gap-2">
          <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm ${step >= 2 ? 'bg-blue-500 text-white' : 'bg-gray-800 text-gray-500'}`}>2</div>
          <span className={`text-xs uppercase tracking-wider font-semibold ${step >= 2 ? 'text-blue-400' : 'text-gray-500'}`}>Join Team</span>
        </div>
        <div className="flex flex-col items-center gap-2">
          <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm ${step >= 3 ? 'bg-blue-500 text-white' : 'bg-gray-800 text-gray-500'}`}>3</div>
          <span className={`text-xs uppercase tracking-wider font-semibold ${step >= 3 ? 'text-blue-400' : 'text-gray-500'}`}>Ready!</span>
        </div>
      </div>

      <div className="max-w-2xl mx-auto">
        {step === 1 && (
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-8 text-center shadow-xl">
            <h2 className="text-2xl font-bold text-white mb-4">Register for {event.title}</h2>
            <p className="text-gray-400 mb-8 max-w-md mx-auto">
              Join this hackathon to start forming your team and submitting your projects.
            </p>
            <button
              onClick={handleRegister}
              disabled={isRegistering || event.status !== 'registration_open'}
              className="bg-blue-600 hover:bg-blue-500 text-white font-medium py-3 px-8 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isRegistering ? 'Registering...' : 'Register Now'}
            </button>
            {event.status !== 'registration_open' && (
              <p className="text-yellow-500 text-sm mt-4">Registration is currently not open for this event.</p>
            )}
          </div>
        )}

        {step === 2 && !isLoadingTeam && (
          <div className="space-y-8">
            <CreateTeamForm eventSlug={slug} onSuccess={refetchTeam} />
            
            <div className="flex items-center gap-4">
              <div className="h-px bg-gray-800 flex-1" />
              <span className="text-gray-500 text-sm font-semibold uppercase tracking-wider">OR</span>
              <div className="h-px bg-gray-800 flex-1" />
            </div>

            <JoinWithCodeForm eventSlug={slug} onSuccess={refetchTeam} />
          </div>
        )}

        {step === 3 && team && (
          <div>
            <div className="mb-6 flex items-center justify-between">
              <h2 className="text-2xl font-bold text-white">Your Team</h2>
              <button 
                onClick={() => router.push(`/events/${slug}/submissions/new`)}
                className="bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
              >
                Start Submission
              </button>
            </div>
            <TeamCard team={team} onLeave={handleLeaveTeam} currentUserId={user?.id} />
          </div>
        )}

        {isLoadingTeam && step > 1 && (
          <div className="text-center p-12 text-gray-400">Loading team...</div>
        )}
      </div>
    </div>
  );
}
