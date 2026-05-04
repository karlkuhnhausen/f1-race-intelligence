import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import StandingsPage from '../../src/features/standings/StandingsPage';
import type {
  DriversStandingsResponse,
  ConstructorsStandingsResponse,
} from '../../src/features/standings/standingsApi';

const mockDrivers: DriversStandingsResponse = {
  year: 2026,
  data_as_of_utc: '2026-04-19T12:00:00Z',
  rows: [
    { position: 1, driver_number: 1, driver_name: 'Max Verstappen', team_name: 'Red Bull Racing', team_color: '3671C6', points: 119, wins: 4, podiums: 5, dnfs: 0, poles: 3 },
    { position: 2, driver_number: 4, driver_name: 'Lando Norris', team_name: 'McLaren', team_color: 'FF8000', points: 98, wins: 2, podiums: 4, dnfs: 1, poles: 1 },
    { position: 3, driver_number: 16, driver_name: 'Charles Leclerc', team_name: 'Ferrari', team_color: 'E8002D', points: 87, wins: 1, podiums: 3, dnfs: 0, poles: 2 },
  ],
};

const mockConstructors: ConstructorsStandingsResponse = {
  year: 2026,
  data_as_of_utc: '2026-04-19T12:00:00Z',
  rows: [
    { position: 1, team_name: 'Red Bull Racing', team_color: '3671C6', points: 198, wins: 4, podiums: 8, dnfs: 0 },
    { position: 2, team_name: 'McLaren', team_color: 'FF8000', points: 165, wins: 2, podiums: 6, dnfs: 1 },
    { position: 3, team_name: 'Ferrari', team_color: 'E8002D', points: 150, wins: 1, podiums: 5, dnfs: 0 },
  ],
};

vi.mock('../../src/features/standings/standingsApi', () => ({
  fetchDriverStandings: vi.fn(() => Promise.resolve(mockDrivers)),
  fetchConstructorStandings: vi.fn(() => Promise.resolve(mockConstructors)),
  fetchDriverProgression: vi.fn(() => Promise.resolve({ year: 2026, rounds: ['Round 1'], drivers: [] })),
  fetchConstructorProgression: vi.fn(() => Promise.resolve({ year: 2026, rounds: ['Round 1'], teams: [] })),
}));

describe('StandingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders driver standings by default', async () => {
    render(<StandingsPage />);

    expect(await screen.findByText('Max Verstappen')).toBeDefined();
    expect(screen.getByText('Lando Norris')).toBeDefined();
    expect(screen.getByText('Charles Leclerc')).toBeDefined();
  });

  it('renders driver position, points, and wins', async () => {
    render(<StandingsPage />);

    await screen.findByText('Max Verstappen');

    // Team is conveyed via color accent border, not as a text column,
    // so we only assert the data columns now present in StandingsTable.
    expect(screen.getByText('119')).toBeDefined();
    expect(screen.getAllByText('4').length).toBeGreaterThan(0);
  });

  it('switches to constructors tab', async () => {
    render(<StandingsPage />);

    await screen.findByText('Max Verstappen');

    fireEvent.click(screen.getByText('Constructors'));

    expect(screen.getByText('Red Bull Racing')).toBeDefined();
    expect(screen.getByText('198')).toBeDefined();
    expect(screen.getByText('McLaren')).toBeDefined();
    expect(screen.getByText('165')).toBeDefined();
  });

  it('shows loading state initially', () => {
    render(<StandingsPage />);
    expect(screen.getByText('Loading standings…')).toBeDefined();
  });

  it('renders team color as left border accent', async () => {
    render(<StandingsPage />);

    await screen.findByText('Max Verstappen');

    const rows = screen.getAllByTestId('standings-row');
    // First row should have Red Bull's team color as border
    expect(rows[0].style.borderLeft).toContain('rgb(54, 113, 198)');
  });

  it('renders zero stats as "0" not blank', async () => {
    render(<StandingsPage />);

    await screen.findByText('Max Verstappen');

    // Verstappen has 0 DNFs — should render as "0" in the DNFs column
    const rows = screen.getAllByTestId('standings-row');
    const firstRowCells = rows[0].querySelectorAll('td');
    // Columns: Pos, Name, Points, Wins, Podiums, DNFs, Poles
    // Index 5 = DNFs column
    expect(firstRowCells[5].textContent).toBe('0');
  });

  it('renders year picker with current year selected', async () => {
    render(<StandingsPage />);

    await screen.findByText('Max Verstappen');

    const picker = screen.getByTestId('year-picker') as HTMLSelectElement;
    expect(picker.value).toBe(String(new Date().getFullYear()));
  });
});
