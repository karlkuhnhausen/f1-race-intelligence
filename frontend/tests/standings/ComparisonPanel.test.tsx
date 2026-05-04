import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import ComparisonPanel from '../../src/features/standings/ComparisonPanel';
import type { DriverComparisonResponse } from '../../src/features/standings/standingsApi';

const mockComparison: DriverComparisonResponse = {
  year: 2025,
  driver1: {
    driver_number: 1,
    driver_name: 'Max Verstappen',
    team_name: 'Red Bull Racing',
    team_color: '3671C6',
    points: 119,
    wins: 4,
    podiums: 5,
    dnfs: 0,
    poles: 3,
  },
  driver2: {
    driver_number: 4,
    driver_name: 'Lando Norris',
    team_name: 'McLaren',
    team_color: 'FF8000',
    points: 98,
    wins: 2,
    podiums: 4,
    dnfs: 1,
    poles: 1,
  },
  deltas: { points: 21, wins: 2, podiums: 1, dnfs: -1, poles: 2 },
  rounds: ['Round 1', 'Round 2', 'Round 3'],
  driver1_points: [25, 50, 119],
  driver2_points: [18, 43, 98],
};

describe('ComparisonPanel', () => {
  it('renders comparison with stats and deltas', () => {
    render(<ComparisonPanel data={mockComparison} />);

    expect(screen.getByTestId('comparison-panel')).toBeDefined();
    expect(screen.getByText(/Max Verstappen vs Lando Norris/)).toBeDefined();
    expect(screen.getByText('+21 pts')).toBeDefined();
    expect(screen.getByText('+2 wins')).toBeDefined();
  });

  it('renders nothing when data is null', () => {
    const { container } = render(<ComparisonPanel data={null} />);
    expect(container.innerHTML).toBe('');
  });

  it('shows loading state', () => {
    render(<ComparisonPanel data={null} loading={true} />);
    expect(screen.getByText(/Loading comparison/)).toBeDefined();
  });
});
