import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import ConstructorBreakdown from '../../src/features/standings/ConstructorBreakdown';

vi.mock('../../src/features/standings/standingsApi', () => ({
  fetchConstructorDriverBreakdown: vi.fn(() =>
    Promise.resolve({
      year: 2025,
      team_name: 'Red Bull Racing',
      team_points: 200,
      drivers: [
        { driver_number: 1, driver_name: 'Max Verstappen', position: 1, points: 150, wins: 4, podiums: 5, points_percentage: 75 },
        { driver_number: 11, driver_name: 'Sergio Perez', position: 8, points: 50, wins: 0, podiums: 2, points_percentage: 25 },
      ],
    }),
  ),
}));

describe('ConstructorBreakdown', () => {
  it('renders driver breakdown with percentages', async () => {
    render(<ConstructorBreakdown teamName="Red Bull Racing" year={2025} />);

    expect(await screen.findByText('Max Verstappen')).toBeDefined();
    expect(screen.getByText('Sergio Perez')).toBeDefined();
    expect(screen.getByText('75%')).toBeDefined();
    expect(screen.getByText('25%')).toBeDefined();
  });

  it('renders driver points that sum to team total', async () => {
    render(<ConstructorBreakdown teamName="Red Bull Racing" year={2025} />);

    await screen.findByText('Max Verstappen');

    expect(screen.getByText('150')).toBeDefined();
    expect(screen.getByText('50')).toBeDefined();
  });
});
