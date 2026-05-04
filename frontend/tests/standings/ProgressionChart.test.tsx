import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import ProgressionChart from '../../src/features/standings/ProgressionChart';

describe('ProgressionChart', () => {
  const mockEntries = [
    { name: 'Verstappen', color: '3671C6', pointsByRound: [25, 43, 69] },
    { name: 'Norris', color: 'FF8000', pointsByRound: [18, 36, 61] },
  ];

  it('renders chart container with multiple rounds', () => {
    render(
      <ProgressionChart
        rounds={['Round 1', 'Round 2', 'Round 3']}
        entries={mockEntries}
      />,
    );

    expect(screen.getByTestId('progression-chart')).toBeDefined();
  });

  it('shows empty state message with single round', () => {
    render(
      <ProgressionChart
        rounds={['Round 1']}
        entries={[{ name: 'Verstappen', color: '3671C6', pointsByRound: [25] }]}
      />,
    );

    expect(screen.getByText(/not enough data/i)).toBeDefined();
  });

  it('renders with empty entries array', () => {
    render(
      <ProgressionChart
        rounds={['Round 1', 'Round 2']}
        entries={[]}
      />,
    );

    expect(screen.getByTestId('progression-chart')).toBeDefined();
  });
});
