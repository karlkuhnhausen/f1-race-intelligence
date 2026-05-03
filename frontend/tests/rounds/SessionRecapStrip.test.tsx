import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import SessionRecapStrip from '../../src/features/rounds/SessionRecapStrip';
import type { SessionDetail } from '../../src/features/rounds/roundApi';

function makeSession(
  type: string,
  name: string,
  status: string,
  dateStart: string,
  recap?: SessionDetail['recap_summary'],
): SessionDetail {
  return {
    session_name: name,
    session_type: type,
    status,
    date_start_utc: dateStart,
    date_end_utc: dateStart,
    results: [],
    recap_summary: recap,
  };
}

const raceRecap = {
  winner_name: 'Max Verstappen',
  winner_team: 'Red Bull Racing',
};

const qualifyingRecap = {
  pole_sitter_name: 'Charles Leclerc',
  pole_sitter_team: 'Ferrari',
};

const practiceRecap = {
  best_driver_name: 'Lando Norris',
  best_driver_team: 'McLaren',
};

describe('SessionRecapStrip', () => {
  it('renders null when no completed sessions', () => {
    const sessions = [
      makeSession('race', 'Race', 'upcoming', '2026-04-06T13:00:00Z'),
    ];
    const { container } = render(<SessionRecapStrip sessions={sessions} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders null when completed sessions have no recap_summary', () => {
    const sessions = [
      makeSession('race', 'Race', 'completed', '2026-04-06T13:00:00Z'),
    ];
    const { container } = render(<SessionRecapStrip sessions={sessions} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders one card per completed session with recap_summary', () => {
    const sessions = [
      makeSession('race', 'Race', 'completed', '2026-04-06T13:00:00Z', raceRecap),
      makeSession('qualifying', 'Qualifying', 'completed', '2026-04-05T14:00:00Z', qualifyingRecap),
    ];
    render(<SessionRecapStrip sessions={sessions} />);
    expect(screen.getByText('Max Verstappen')).toBeDefined();
    expect(screen.getByText('Charles Leclerc')).toBeDefined();
  });

  it('renders cards in chronological order by date_start_utc', () => {
    const sessions = [
      makeSession('race', 'Race', 'completed', '2026-04-06T13:00:00Z', raceRecap),
      makeSession('practice1', 'Practice 1', 'completed', '2026-04-04T11:00:00Z', practiceRecap),
    ];
    render(<SessionRecapStrip sessions={sessions} />);
    const practiceLabel = screen.getByText('Practice 1');
    const raceLabel = screen.getByText('Race');
    // Practice should appear before Race in the DOM
    expect(
      practiceLabel.compareDocumentPosition(raceLabel) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it('dispatches race session type to RaceRecapCard (shows winner_name)', () => {
    const sessions = [
      makeSession('race', 'Race', 'completed', '2026-04-06T13:00:00Z', raceRecap),
    ];
    render(<SessionRecapStrip sessions={sessions} />);
    expect(screen.getByText('Max Verstappen')).toBeDefined();
  });

  it('dispatches sprint session type to RaceRecapCard', () => {
    const sessions = [
      makeSession('sprint', 'Sprint', 'completed', '2026-04-05T10:00:00Z', raceRecap),
    ];
    render(<SessionRecapStrip sessions={sessions} />);
    expect(screen.getByText('Max Verstappen')).toBeDefined();
  });

  it('dispatches qualifying session to QualifyingRecapCard (shows pole_sitter_name)', () => {
    const sessions = [
      makeSession('qualifying', 'Qualifying', 'completed', '2026-04-05T14:00:00Z', qualifyingRecap),
    ];
    render(<SessionRecapStrip sessions={sessions} />);
    expect(screen.getByText('Charles Leclerc')).toBeDefined();
  });

  it('dispatches sprint_qualifying to QualifyingRecapCard', () => {
    const sessions = [
      makeSession('sprint_qualifying', 'Sprint Qualifying', 'completed', '2026-04-05T09:00:00Z', qualifyingRecap),
    ];
    render(<SessionRecapStrip sessions={sessions} />);
    expect(screen.getByText('Charles Leclerc')).toBeDefined();
  });

  it('dispatches practice session to PracticeRecapCard (shows best_driver_name)', () => {
    const sessions = [
      makeSession('practice1', 'Practice 1', 'completed', '2026-04-04T11:00:00Z', practiceRecap),
    ];
    render(<SessionRecapStrip sessions={sessions} />);
    expect(screen.getByText('Lando Norris')).toBeDefined();
  });
});
