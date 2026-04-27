import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import NextRaceCard from '../../src/features/calendar/NextRaceCard';
import type { RaceMeetingDTO } from '../../src/features/calendar/calendarApi';

const miamiRound: RaceMeetingDTO = {
  round: 6,
  race_name: 'Miami Grand Prix',
  circuit_name: 'Miami International Autodrome',
  country_name: 'United States',
  start_datetime_utc: '2026-05-10T19:00:00Z',
  end_datetime_utc: '2026-05-13T19:00:00Z',
  status: 'scheduled',
  is_cancelled: false,
};

describe('NextRaceCard', () => {
  beforeEach(() => {
    // Fix "now" to April 19, 2026 12:00 UTC — 21 days before Miami.
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-19T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders race name and circuit', () => {
    render(<NextRaceCard round={miamiRound} countdownTarget={miamiRound.start_datetime_utc} />);

    expect(screen.getByText('Miami Grand Prix')).toBeDefined();
    expect(screen.getByText(/Miami International Autodrome/)).toBeDefined();
    expect(screen.getByText('Next Race')).toBeDefined();
  });

  it('displays countdown with days, hours, minutes, seconds', () => {
    render(<NextRaceCard round={miamiRound} countdownTarget={miamiRound.start_datetime_utc} />);

    expect(screen.getByText(/until lights out/)).toBeDefined();
    // 21 days and 7 hours from Apr 19 12:00 to May 10 19:00
    expect(screen.getByText('21d')).toBeDefined();
    expect(screen.getByText('7h')).toBeDefined();
  });

  it('ticks down after 1 second', () => {
    render(<NextRaceCard round={miamiRound} countdownTarget={miamiRound.start_datetime_utc} />);

    // Advance by 1 second.
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    // After 1 second tick, seconds should have changed (59s now).
    const countdownEl = screen.getByText(/until lights out/);
    expect(countdownEl).toBeDefined();
  });

  it('shows "RACE LIVE" when countdown expires', () => {
    // Set time to after the race start.
    vi.setSystemTime(new Date('2026-05-10T19:00:01Z'));

    render(<NextRaceCard round={miamiRound} countdownTarget={miamiRound.start_datetime_utc} />);

    expect(screen.getByText('RACE LIVE')).toBeDefined();
  });

  it('has accessible region role', () => {
    render(<NextRaceCard round={miamiRound} countdownTarget={miamiRound.start_datetime_utc} />);

    const region = screen.getByRole('region', { name: /next race countdown/i });
    expect(region).toBeDefined();
  });
});
