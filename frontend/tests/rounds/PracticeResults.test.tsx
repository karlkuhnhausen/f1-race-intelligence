import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import PracticeResults from '../../src/features/rounds/PracticeResults';
import type { SessionResultEntry } from '../../src/features/rounds/roundApi';

const practiceResults: SessionResultEntry[] = [
  {
    position: 1,
    driver_number: 1,
    driver_name: 'Max VERSTAPPEN',
    driver_acronym: 'VER',
    team_name: 'Red Bull Racing',
    number_of_laps: 24,
    best_lap_time: 77.456,
    gap_to_fastest: 0,
  },
  {
    position: 2,
    driver_number: 16,
    driver_name: 'Charles LECLERC',
    driver_acronym: 'LEC',
    team_name: 'Ferrari',
    number_of_laps: 22,
    best_lap_time: 77.789,
    gap_to_fastest: 0.333,
  },
  {
    position: 3,
    driver_number: 44,
    driver_name: 'Lewis HAMILTON',
    driver_acronym: 'HAM',
    team_name: 'Ferrari',
    number_of_laps: 20,
    best_lap_time: 78.012,
    gap_to_fastest: 0.556,
  },
];

describe('PracticeResults', () => {
  it('renders position, driver, team, best lap, and lap count', () => {
    render(<PracticeResults results={practiceResults} />);

    expect(screen.getByText('Max VERSTAPPEN')).toBeDefined();
    expect(screen.getByText('VER')).toBeDefined();
    expect(screen.getByText('Red Bull Racing')).toBeDefined();
    expect(screen.getByText('24')).toBeDefined();
    expect(screen.getByText('22')).toBeDefined();
  });

  it('formats best lap time as M:SS.sss', () => {
    render(<PracticeResults results={practiceResults} />);

    // 77.456 → 1:17.456
    expect(screen.getByText('1:17.456')).toBeDefined();
    // 77.789 → 1:17.789
    expect(screen.getByText('1:17.789')).toBeDefined();
  });

  it('shows em-dash for the fastest driver gap and "+x.xxxs" for others', () => {
    render(<PracticeResults results={practiceResults} />);

    expect(screen.getByText('+0.333s')).toBeDefined();
    expect(screen.getByText('+0.556s')).toBeDefined();
    // Verstappen (gap 0) shows '—'
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(1);
  });

  it('shows "Not yet available" when results array is empty', () => {
    render(<PracticeResults results={[]} />);
    expect(screen.getByText('Not yet available')).toBeDefined();
  });
});
