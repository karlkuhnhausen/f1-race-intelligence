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
    { position: 1, driver_name: 'Max Verstappen', team_name: 'Red Bull Racing', points: 119, wins: 4 },
    { position: 2, driver_name: 'Lando Norris', team_name: 'McLaren', points: 98, wins: 2 },
    { position: 3, driver_name: 'Charles Leclerc', team_name: 'Ferrari', points: 87, wins: 1 },
  ],
};

const mockConstructors: ConstructorsStandingsResponse = {
  year: 2026,
  data_as_of_utc: '2026-04-19T12:00:00Z',
  rows: [
    { position: 1, team_name: 'Red Bull Racing', points: 198 },
    { position: 2, team_name: 'McLaren', points: 165 },
    { position: 3, team_name: 'Ferrari', points: 150 },
  ],
};

vi.mock('../../src/features/standings/standingsApi', () => ({
  fetchDriverStandings: vi.fn(() => Promise.resolve(mockDrivers)),
  fetchConstructorStandings: vi.fn(() => Promise.resolve(mockConstructors)),
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
    expect(screen.getByText('4')).toBeDefined();
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
});
