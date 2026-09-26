'use client';

import React, { useState } from 'react';
import { apiClient } from '../../lib/api';

interface JoinWithCodeFormProps {
  eventSlug: string;
  onSuccess: () => void;
}

export function JoinWithCodeForm({ eventSlug, onSuccess }: JoinWithCodeFormProps) {
  const [code, setCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (code.length !== 8) {
      setError('Invite code must be exactly 8 characters');
      return;
    }
    
    if (!/^[A-Z0-9]{8}$/.test(code)) {
      setError('Invalid code format');
      return;
    }

    setIsSubmitting(true);
    try {
      await apiClient.joinTeam(eventSlug, code);
      onSuccess();
    } catch (err: any) {
      if (err.status === 404) {
        setError('Invalid code');
      } else if (err.status === 422) {
        setError('Team is full');
      } else {
        setError(err.message || 'Failed to join team');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCodeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setCode(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 8));
  };

  return (
    <form onSubmit={handleSubmit} className="bg-gray-900 border border-gray-800 rounded-xl p-6 shadow-xl">
      <h3 className="text-xl font-bold text-white mb-4">Join an Existing Team</h3>
      <p className="text-gray-400 text-sm mb-6">
        Have an invite code? Enter it below to join your friends.
      </p>

      <div className="space-y-4">
        <div>
          <label htmlFor="inviteCode" className="block text-sm font-medium text-gray-300 mb-1">
            Invite Code
          </label>
          <input
            id="inviteCode"
            type="text"
            value={code}
            onChange={handleCodeChange}
            className="w-full bg-gray-800 border border-gray-700 text-white rounded-lg px-4 py-2 font-mono text-center tracking-widest focus:outline-none focus:ring-2 focus:ring-teal-500 transition-shadow"
            placeholder="ABCDEFGH"
            disabled={isSubmitting}
            maxLength={8}
          />
          {error && <p className="text-red-400 text-sm mt-2">{error}</p>}
        </div>

        <button
          type="submit"
          disabled={isSubmitting || code.length !== 8}
          className="w-full bg-teal-600 hover:bg-teal-500 text-white font-medium py-2 px-4 rounded-lg transition-colors disabled:opacity-50"
        >
          {isSubmitting ? 'Joining...' : 'Join Team'}
        </button>
      </div>
    </form>
  );
}
