import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import RaceRecapCard from '../../src/features/rounds/RaceRecapCard';
import QualifyingRecapCard from '../../src/features/rounds/QualifyingRecapCard';
import PracticeRecapCard from '../../src/features/rounds/PracticeRecapCard';
import type { SessionDetail } from '../../src/features/rounds/roundApi';

function makeSession(overrides: Partial<SessionDetail> = {}): SessionDetail {
  return {
    session_name: 'Race',
    session_type: 'race',
    status: 'completed',
    date_start_utc: '2026-04-06T13:00:00Z',
    date_end_utc: '2026-04-06T15:00:00Z',
    results: [],
    ...overrides,
  };
}

describe('RaceRecapCard', () => {
  it('renders winner name and team', () => {
    const session = makeSession({
      recap_summary: {
        winner_name: 'Max Verstappen',
        winner_team: 'Red Bull Racing',
      },
    });
    render(<RaceRecapCard session={session} />);
    expect(screen.getByText('Max Verstappen')).toBeDefined();
    expect(screen.getByText('Red Bull Racing')).toBeDefined();
  });

  it('renders gap to P2 when present', () => {
    const session = makeSession({
      recap_summary: {
        winner_name: 'Max Verstappen',
        winner_team: 'Red Bull Racing',
        gap_to_p2: '+8.294',
      },
    });
    render(<RaceRecapCard session={session} />);
    expect(screen.getByText('+8.294')).toBeDefined();
  });

  it('omits fastest lap row when absent', () => {
    const session = makeSession({
      recap_summary: { winner_name: 'Max Verstappen', winner_team: 'Red Bull Racing' },
    });
    render(<RaceRecapCard session={session} />);
    expect(screen.queryByText('Fastest Lap')).toBeNull();
  });

  it('renders safety car top event', () => {
    const session = makeSession({
      recap_summary: {
        winner_name: 'Max Verstappen',
        winner_team: 'Red Bull Racing',
        top_event: { event_type: 'safety_car', lap_number: 14, count: 2 },
      },
    });
    render(<RaceRecapCard session={session} />);
    expect(screen.getByText(/Safety Car/)).toBeDefined();
  });

  it('renders red flag top event', () => {
    const session = makeSession({
      recap_summary: {
        winner_name: 'Max Verstappen',
        winner_team: 'Red Bull Racing',
        top_event: { event_type: 'red_flag', lap_number: 5, count: 1 },
      },
    });
    render(<RaceRecapCard session={session} />);
    expect(screen.getByText(/Red Flag/)).toBeDefined();
  });

  it('omits top event row when no events', () => {
    const session = makeSession({
      recap_summary: { winner_name: 'Max Verstappen', winner_team: 'Red Bull Racing' },
    });
    render(<RaceRecapCard session={session} />);
    expect(screen.queryByText(/Safety Car/)).toBeNull();
    expect(screen.queryByText(/Red Flag/)).toBeNull();
  });
});

describe('QualifyingRecapCard', () => {
  it('renders pole sitter and pole time', () => {
    const session = makeSession({
      session_name: 'Qualifying',
      session_type: 'qualifying',
      recap_summary: {
        pole_sitter_name: 'Charles Leclerc',
        pole_sitter_team: 'Ferrari',
        pole_time: 86.983,
      },
    });
    render(<QualifyingRecapCard session={session} />);
    expect(screen.getByText('Charles Leclerc')).toBeDefined();
    expect(screen.getByText('Ferrari')).toBeDefined();
    expect(screen.getByText('Pole Time')).toBeDefined();
  });

  it('renders Q1 and Q2 cutoff rows when present', () => {
    const session = makeSession({
      session_name: 'Qualifying',
      session_type: 'qualifying',
      recap_summary: {
        pole_sitter_name: 'Charles Leclerc',
        pole_sitter_team: 'Ferrari',
        pole_time: 86.983,
        q1_cutoff_time: 88.211,
        q2_cutoff_time: 87.654,
      },
    });
    render(<QualifyingRecapCard session={session} />);
    expect(screen.getByText('Q1 Cutoff')).toBeDefined();
    expect(screen.getByText('Q2 Cutoff')).toBeDefined();
  });

  it('omits Q1/Q2 cutoff rows when absent', () => {
    const session = makeSession({
      session_name: 'Qualifying',
      session_type: 'qualifying',
      recap_summary: {
        pole_sitter_name: 'Charles Leclerc',
        pole_sitter_team: 'Ferrari',
        pole_time: 86.983,
      },
    });
    render(<QualifyingRecapCard session={session} />);
    expect(screen.queryByText('Q1 Cutoff')).toBeNull();
    expect(screen.queryByText('Q2 Cutoff')).toBeNull();
  });

  it('renders red flag count when present', () => {
    const session = makeSession({
      session_name: 'Qualifying',
      session_type: 'qualifying',
      recap_summary: {
        pole_sitter_name: 'Charles Leclerc',
        pole_sitter_team: 'Ferrari',
        red_flag_count: 2,
      },
    });
    render(<QualifyingRecapCard session={session} />);
    expect(screen.getByText(/Red Flag/)).toBeDefined();
  });

  it('omits red flag row when zero', () => {
    const session = makeSession({
      session_name: 'Qualifying',
      session_type: 'qualifying',
      recap_summary: {
        pole_sitter_name: 'Charles Leclerc',
        pole_sitter_team: 'Ferrari',
      },
    });
    render(<QualifyingRecapCard session={session} />);
    expect(screen.queryByText(/Red Flag/)).toBeNull();
  });
});

describe('PracticeRecapCard', () => {
  it('renders best driver and best lap time label', () => {
    const session = makeSession({
      session_name: 'Practice 1',
      session_type: 'practice1',
      recap_summary: {
        best_driver_name: 'Lando Norris',
        best_driver_team: 'McLaren',
        best_lap_time: 88.5,
      },
    });
    render(<PracticeRecapCard session={session} />);
    expect(screen.getByText('Lando Norris')).toBeDefined();
    expect(screen.getByText('McLaren')).toBeDefined();
    expect(screen.getByText('Best Lap')).toBeDefined();
  });

  it('renders total laps when present', () => {
    const session = makeSession({
      session_name: 'Practice 1',
      session_type: 'practice1',
      recap_summary: {
        best_driver_name: 'Lando Norris',
        best_driver_team: 'McLaren',
        total_laps: 420,
      },
    });
    render(<PracticeRecapCard session={session} />);
    expect(screen.getByText('420 laps')).toBeDefined();
  });

  it('omits red flag row when zero', () => {
    const session = makeSession({
      session_name: 'Practice 1',
      session_type: 'practice1',
      recap_summary: {
        best_driver_name: 'Lando Norris',
        best_driver_team: 'McLaren',
      },
    });
    render(<PracticeRecapCard session={session} />);
    expect(screen.queryByText(/Red Flag/)).toBeNull();
  });
});
