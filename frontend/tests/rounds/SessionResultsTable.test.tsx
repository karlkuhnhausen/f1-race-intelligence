import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import SessionResultsTable from '../../src/features/rounds/SessionResultsTable';
import type { SessionResultEntry } from '../../src/features/rounds/roundApi';

const raceResults: SessionResultEntry[] = [
  {
    position: 1,
    driver_number: 1,
    driver_name: 'Max VERSTAPPEN',
    driver_acronym: 'VER',
    team_name: 'Red Bull Racing',
    number_of_laps: 58,
    finishing_status: 'Finished',
    points: 25,
  },
  {
    position: 2,
    driver_number: 16,
    driver_name: 'Charles LECLERC',
    driver_acronym: 'LEC',
    team_name: 'Ferrari',
    number_of_laps: 58,
    finishing_status: 'Finished',
    gap_to_leader: '+5.123s',
    points: 18,
  },
  {
    position: 19,
    driver_number: 27,
    driver_name: 'Nico HULKENBERG',
    driver_acronym: 'HUL',
    team_name: 'Haas',
    number_of_laps: 32,
    finishing_status: 'DNF',
  },
];

describe('SessionResultsTable', () => {
  it('renders a team color swatch with a background-color style', () => {
    render(<SessionResultsTable results={raceResults} sessionType="race" />);
    const swatches = screen.getAllByTestId('team-color-swatch');
    expect(swatches.length).toBe(raceResults.length);
    // Red Bull resolves to #3671c6 via teamColors normalization.
    expect(swatches[0].getAttribute('style')).toContain('rgb(54, 113, 198)');
  });

  it('renders the Not Classified divider for DNF rows in race sessions', () => {
    render(<SessionResultsTable results={raceResults} sessionType="race" />);
    expect(screen.getByText('Not Classified')).toBeDefined();
    // DNF row is still rendered after the divider.
    expect(screen.getByText('Nico HULKENBERG')).toBeDefined();
  });

  it('uses tabular-nums for numeric columns (Pos, #, Gap, Pts)', () => {
    const { container } = render(
      <SessionResultsTable results={raceResults} sessionType="race" />
    );
    const tabularCells = container.querySelectorAll('td.tabular-nums');
    // Each row contributes Pos (#), driver_number, gap, pts = 4 tabular cells
    // for race; HULKENBERG (DNF) has the same numeric columns.
    expect(tabularCells.length).toBeGreaterThanOrEqual(4 * raceResults.length);
  });

  it('does not render Not Classified divider when there are no DNFs', () => {
    const allFinished = raceResults.slice(0, 2);
    render(<SessionResultsTable results={allFinished} sessionType="race" />);
    expect(screen.queryByText('Not Classified')).toBeNull();
  });

  it('does not render Not Classified divider for non-race sessions', () => {
    const qualiResults: SessionResultEntry[] = [
      {
        position: 1,
        driver_number: 1,
        driver_name: 'Max VERSTAPPEN',
        driver_acronym: 'VER',
        team_name: 'Red Bull Racing',
        number_of_laps: 0,
        q1_time: 78.0,
        q2_time: 77.0,
        q3_time: 76.5,
      },
    ];
    render(<SessionResultsTable results={qualiResults} sessionType="qualifying" />);
    expect(screen.queryByText('Not Classified')).toBeNull();
  });
});
