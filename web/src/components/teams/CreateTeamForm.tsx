'use client';

import React, { useState } from 'react';
import { apiClient } from '../../lib/api';

interface CreateTeamFormProps {
  eventSlug: string;
  onSuccess: () => void;
}

export function CreateTeamForm({ eventSlug, onSuccess }: CreateTeamFormProps) {
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!name.trim()) {
      setError('Team name is required');
      return;
    }

    setIsSubmitting(true);
    try {
      await apiClient.createTeam(eventSlug, name.trim());
      onSuccess();
    } catch (err: any) {
      if (err.status === 409) {
        setError('Team name taken');
      } else {
        setError(err.message || 'Failed to create team');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="bg-gray-900 border border-gray-800 rounded-xl p-6 shadow-xl">
      <h3 className="text-xl font-bold text-white mb-4">Create a New Team</h3>
      <p className="text-gray-400 text-sm mb-6">
        Create a team to get an invite code you can share with your friends.
      </p>

      <div className="space-y-4">
        <div>
          <label htmlFor="teamName" className="block text-sm font-medium text-gray-300 mb-1">
            Team Name
          </label>
          <input
            id="teamName"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full bg-gray-800 border border-gray-700 text-white rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-shadow"
            placeholder="e.g. The Innovators"
            disabled={isSubmitting}
          />
          {error && <p className="text-red-400 text-sm mt-2">{error}</p>}
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          className="w-full bg-blue-600 hover:bg-blue-500 text-white font-medium py-2 px-4 rounded-lg transition-colors disabled:opacity-50"
        >
          {isSubmitting ? 'Creating...' : 'Create Team'}
        </button>
      </div>
    </form>
  );
}
