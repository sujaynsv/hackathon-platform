export interface Track {
  id: string;
  name: string;
  description?: string;
}

export interface Event {
  id: string;
  slug: string;
  title: string;
  description?: string;
  bannerUrl?: string;
  organizerId: string;
  status: string;
  maxTeamSize: number;
  registrationOpensAt?: string;
  registrationClosesAt?: string;
  submissionDeadlineAt?: string;
  judgingDeadlineAt?: string;
  votingOpensAt?: string;
  votingClosesAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface EventDetail extends Event {
  tracks: Track[];
  myRole?: string;
}

export interface CreateEventRequest {
  slug: string;
  title: string;
  description?: string;
  maxTeamSize: number;
  registrationOpensAt?: string;
  registrationClosesAt?: string;
  submissionDeadlineAt?: string;
  judgingDeadlineAt?: string;
  votingOpensAt?: string;
  votingClosesAt?: string;
}

export interface UpdateEventRequest {
  title?: string;
  description?: string;
  status?: string;
  maxTeamSize?: number;
  registrationOpensAt?: string;
  registrationClosesAt?: string;
  submissionDeadlineAt?: string;
  judgingDeadlineAt?: string;
  votingOpensAt?: string;
  votingClosesAt?: string;
}
