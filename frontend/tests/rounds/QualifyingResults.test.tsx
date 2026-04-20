import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import QualifyingResults from '../../src/features/rounds/QualifyingResults';
import type { SessionResultEntry } from '../../src/features/rounds/roundApi';

const qualifyingResults: SessionResultEntry[] = [
  {
    position: 1,
    driver_number: 1,
    driver_name: 'Max VERSTAPPEN',
    driver_acronym: 'VER',
    team_name: 'Red Bull Racing',
    number_of_laps: 18,
    q1_time: 78.123,
    q2_time: 77.456,
    q3_time: 76.789,
  },
  {
    position: 11,
    driver_number: 44,
    driver_name: 'Lewis HAMILTON',
    driver_acronym: 'HAM',
    team_name: 'Ferrari',
    number_of_laps: 12,
    q1_time: 78.501,
    q2_time: 77.999,
    // Q2-eliminated: no q3_time
  },
  {
    position: 16,
    driver_number: 18,
    driver_name: 'Lance STROLL',
    driver_acronym: 'STR',
    team_name: 'Aston Martin',
    number_of_laps: 6,
    q1_time: 79.250,
    // Q1-eliminated: no q2_time, no q3_time
  },
];

describe('QualifyingResults', () => {
  it('renders all drivers with grid positions, acronyms, and team names', () => {
    render(<QualifyingResults results={qualifyingResults} />);

    expect(screen.getByText('Max VERSTAPPEN')).toBeDefined();
    expect(screen.getByText('VER')).toBeDefined();
    expect(screen.getByText('Lewis HAMILTON')).toBeDefined();
    expect(screen.getByText('Lance STROLL')).toBeDefined();
    expect(screen.getAllByText('Ferrari').length).toBeGreaterThan(0);
  });

  it('formats Q1/Q2/Q3 times as M:SS.sss when over a minute', () => {
    render(<QualifyingResults results={qualifyingResults} />);

    // 78.123s → 1:18.123
    expect(screen.getByText('1:18.123')).toBeDefined();
    // 76.789s → 1:16.789 (pole Q3)
    expect(screen.getByText('1:16.789')).toBeDefined();
  });

  it('shows em-dash for Q2 when driver was Q1-eliminated', () => {
    render(<QualifyingResults results={qualifyingResults} />);

    // Stroll has only q1; q2 and q3 cells should show '—'
    const dashes = screen.getAllByText('—');
    // Stroll: 2 dashes (Q2, Q3); Hamilton: 1 dash (Q3) → 3 total
    expect(dashes.length).toBe(3);
  });

  it('shows "Not yet available" when results array is empty', () => {
    render(<QualifyingResults results={[]} />);
    expect(screen.getByText('Not yet available')).toBeDefined();
  });
});
