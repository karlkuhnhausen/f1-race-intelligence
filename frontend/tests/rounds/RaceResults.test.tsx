import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import RaceResults from '../../src/features/rounds/RaceResults';
import type { SessionResultEntry } from '../../src/features/rounds/roundApi';

const classifiedResults: SessionResultEntry[] = [
  {
    position: 1,
    driver_number: 1,
    driver_name: 'Max VERSTAPPEN',
    driver_acronym: 'VER',
    team_name: 'Red Bull Racing',
    number_of_laps: 58,
    finishing_status: 'Finished',
    race_time: 5523.456,
    points: 25,
    fastest_lap: true,
  },
  {
    position: 2,
    driver_number: 44,
    driver_name: 'Lewis HAMILTON',
    driver_acronym: 'HAM',
    team_name: 'Ferrari',
    number_of_laps: 58,
    finishing_status: 'Finished',
    gap_to_leader: '+5.123s',
    points: 18,
  },
  {
    position: 3,
    driver_number: 16,
    driver_name: 'Charles LECLERC',
    driver_acronym: 'LEC',
    team_name: 'Ferrari',
    number_of_laps: 58,
    finishing_status: 'Finished',
    gap_to_leader: '+12.456s',
    points: 15,
  },
];

const dnfResults: SessionResultEntry[] = [
  {
    position: 18,
    driver_number: 14,
    driver_name: 'Fernando ALONSO',
    driver_acronym: 'ALO',
    team_name: 'Aston Martin',
    number_of_laps: 32,
    finishing_status: 'DNF',
    points: 0,
  },
  {
    position: 19,
    driver_number: 77,
    driver_name: 'Valtteri BOTTAS',
    driver_acronym: 'BOT',
    team_name: 'Sauber',
    number_of_laps: 0,
    finishing_status: 'DNS',
    points: 0,
  },
  {
    position: 20,
    driver_number: 10,
    driver_name: 'Pierre GASLY',
    driver_acronym: 'GAS',
    team_name: 'Alpine',
    number_of_laps: 45,
    finishing_status: 'DSQ',
    points: 0,
  },
];

describe('RaceResults', () => {
  it('renders classified finishers with positions, drivers, teams, and points', () => {
    render(<RaceResults results={classifiedResults} />);

    expect(screen.getByText('Max VERSTAPPEN')).toBeDefined();
    expect(screen.getByText('Lewis HAMILTON')).toBeDefined();
    expect(screen.getByText('Charles LECLERC')).toBeDefined();
    expect(screen.getByText('Red Bull Racing')).toBeDefined();
    expect(screen.getAllByText('Ferrari').length).toBe(2);
    expect(screen.getByText('25')).toBeDefined();
    expect(screen.getByText('18')).toBeDefined();
    expect(screen.getByText('15')).toBeDefined();
  });

  it('shows race time for P1 and gap for other positions', () => {
    render(<RaceResults results={classifiedResults} />);

    // P1 gets formatted race time: 5523.456s = 1:32:03.456
    expect(screen.getByText('1:32:03.456')).toBeDefined();
    // P2 and P3 get gap strings
    expect(screen.getByText('+5.123s')).toBeDefined();
    expect(screen.getByText('+12.456s')).toBeDefined();
  });

  it('displays fastest lap badge on the fastest lap driver', () => {
    render(<RaceResults results={classifiedResults} />);

    const badge = screen.getByTitle('Fastest Lap');
    expect(badge).toBeDefined();
    expect(badge.textContent).toBe('⏱');
  });

  it('highlights fastest lap row', () => {
    const { container } = render(<RaceResults results={classifiedResults} />);

    const fastestRow = container.querySelector('tr.fastest-lap');
    expect(fastestRow).not.toBeNull();
    expect(fastestRow!.textContent).toContain('VER');
  });

  it('separates DNF/DNS/DSQ entries below a divider', () => {
    render(<RaceResults results={[...classifiedResults, ...dnfResults]} />);

    expect(screen.getByText('Not Classified')).toBeDefined();
    expect(screen.getByText('DNF')).toBeDefined();
    expect(screen.getByText('DNS')).toBeDefined();
    expect(screen.getByText('DSQ')).toBeDefined();
  });

  it('renders non-classified drivers with their status instead of gap', () => {
    render(<RaceResults results={[...classifiedResults, ...dnfResults]} />);

    expect(screen.getByText('Fernando ALONSO')).toBeDefined();
    expect(screen.getByText('DNF')).toBeDefined();
    expect(screen.getByText('Valtteri BOTTAS')).toBeDefined();
    expect(screen.getByText('DNS')).toBeDefined();
  });

  it('shows laps completed for each driver', () => {
    render(<RaceResults results={[...classifiedResults, ...dnfResults]} />);

    // All classified did 58 laps
    const cells58 = screen.getAllByText('58');
    expect(cells58.length).toBe(3);
    // DNF did 32 laps
    expect(screen.getByText('32')).toBeDefined();
    // DNS did 0 laps, plus points=0 for non-classified drivers
    expect(screen.getAllByText('0').length).toBeGreaterThanOrEqual(1);
  });

  it('renders table headers correctly', () => {
    render(<RaceResults results={classifiedResults} />);

    expect(screen.getByText('Pos')).toBeDefined();
    expect(screen.getByText('Driver')).toBeDefined();
    expect(screen.getByText('Team')).toBeDefined();
    expect(screen.getByText('Time / Gap')).toBeDefined();
    expect(screen.getByText('Laps')).toBeDefined();
    expect(screen.getByText('Pts')).toBeDefined();
  });

  it('renders empty table body when no results', () => {
    const { container } = render(<RaceResults results={[]} />);

    const rows = container.querySelectorAll('tbody tr');
    expect(rows.length).toBe(0);
  });
});
