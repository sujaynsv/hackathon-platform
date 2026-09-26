import { useEffect, useState } from 'react';
import { apiClient } from '../lib/api';
import type { Team } from '../types/api';

export function useMyTeam(eventSlug: string) {
  const [data, setData] = useState<Team | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchTeam = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const team = await apiClient.getMyTeam(eventSlug);
      setData(team);
    } catch (err: any) {
      if (err.status !== 404) {
        setError(err);
      } else {
        setData(null);
      }
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (eventSlug) {
      fetchTeam();
    }
  }, [eventSlug]);

  return { data, isLoading, error, refetch: fetchTeam };
}
