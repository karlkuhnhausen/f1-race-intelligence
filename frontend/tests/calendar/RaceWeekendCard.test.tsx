import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import RaceWeekendCard from '../../src/features/calendar/RaceWeekendCard';
import type {
  ActiveSessionDTO,
  RaceMeetingDTO,
} from '../../src/features/calendar/calendarApi';

const miamiRound: RaceMeetingDTO = {
  round: 6,
  race_name: 'Miami Grand Prix',
  circuit_name: 'Miami International Autodrome',
  country_name: 'United States',
  start_datetime_utc: '2026-05-01T19:00:00Z',
  end_datetime_utc: '2026-05-04T19:00:00Z',
  status: 'scheduled',
  is_cancelled: false,
};

describe('RaceWeekendCard', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders RACE WEEKEND eyebrow and race name', () => {
    vi.setSystemTime(new Date('2026-05-02T15:00:00Z'));
    const session: ActiveSessionDTO = {
      session_type: 'practice3',
      session_name: 'Practice 3',
      status: 'upcoming',
      date_start_utc: '2026-05-02T16:30:00Z',
      date_end_utc: '2026-05-02T17:30:00Z',
    };

    render(<RaceWeekendCard round={miamiRound} session={session} />);

    expect(screen.getByText('RACE WEEKEND')).toBeDefined();
    expect(screen.getByText('Miami Grand Prix')).toBeDefined();
    expect(screen.getByText('Practice 3')).toBeDefined();
    expect(screen.getByText('Up next')).toBeDefined();
    expect(screen.getByText(/until practice 3/i)).toBeDefined();
  });

  it('shows LIVE eyebrow and counts down to session end while in_progress', () => {
    vi.setSystemTime(new Date('2026-05-02T20:30:00Z'));
    const session: ActiveSessionDTO = {
      session_type: 'qualifying',
      session_name: 'Qualifying',
      status: 'in_progress',
      date_start_utc: '2026-05-02T20:00:00Z',
      date_end_utc: '2026-05-02T21:00:00Z',
    };

    render(<RaceWeekendCard round={miamiRound} session={session} />);

    expect(screen.getByText('RACE WEEKEND · LIVE')).toBeDefined();
    expect(screen.getByText('Qualifying')).toBeDefined();
    expect(screen.getByText('Now')).toBeDefined();
    expect(screen.getByText(/qualifying live/i)).toBeDefined();
  });

  it('shows complete state with no countdown when session is completed', () => {
    vi.setSystemTime(new Date('2026-05-03T22:30:00Z'));
    const session: ActiveSessionDTO = {
      session_type: 'race',
      session_name: 'Race',
      status: 'completed',
      date_start_utc: '2026-05-03T19:00:00Z',
      date_end_utc: '2026-05-03T21:00:00Z',
    };

    render(<RaceWeekendCard round={miamiRound} session={session} />);

    expect(screen.getByText('Just finished')).toBeDefined();
    expect(screen.getByText('Race')).toBeDefined();
    expect(screen.getByText(/race complete/i)).toBeDefined();
    // No countdown timer rendered for completed sessions.
    expect(screen.queryByTestId('race-countdown')).toBeNull();
  });

  it('exposes accessible region role', () => {
    vi.setSystemTime(new Date('2026-05-02T15:00:00Z'));
    const session: ActiveSessionDTO = {
      session_type: 'practice3',
      session_name: 'Practice 3',
      status: 'upcoming',
      date_start_utc: '2026-05-02T16:30:00Z',
      date_end_utc: '2026-05-02T17:30:00Z',
    };

    render(<RaceWeekendCard round={miamiRound} session={session} />);
    const region = screen.getByRole('region', { name: /race weekend status/i });
    expect(region).toBeDefined();
  });
});
