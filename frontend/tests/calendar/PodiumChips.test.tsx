import { describe, it, expect } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import PodiumChips from '../../src/features/calendar/PodiumChips';
import type { PodiumEntryDTO } from '../../src/features/calendar/calendarApi';

const podium: PodiumEntryDTO[] = [
  {
    position: 1,
    driver_number: 1,
    driver_acronym: 'VER',
    driver_name: 'Max Verstappen',
    team_name: 'Red Bull Racing',
    season_points: 340,
  },
  {
    position: 2,
    driver_number: 44,
    driver_acronym: 'HAM',
    driver_name: 'Lewis Hamilton',
    team_name: 'Ferrari',
    season_points: 285,
  },
  {
    position: 3,
    driver_number: 63,
    driver_acronym: 'RUS',
    driver_name: 'George Russell',
    team_name: 'Mercedes',
    season_points: 210,
  },
];

describe('PodiumChips', () => {
  it('renders three chips with acronyms and points', () => {
    render(<PodiumChips entries={podium} />);
    const container = screen.getByTestId('podium-chips');
    expect(within(container).getByText('VER')).toBeDefined();
    expect(within(container).getByText('HAM')).toBeDefined();
    expect(within(container).getByText('RUS')).toBeDefined();
    expect(within(container).getByText('340')).toBeDefined();
    expect(within(container).getByText('285')).toBeDefined();
    expect(within(container).getByText('210')).toBeDefined();
  });

  it('labels each chip with P1/P2/P3', () => {
    render(<PodiumChips entries={podium} />);
    expect(screen.getByTestId('podium-chip-p1')).toBeDefined();
    expect(screen.getByTestId('podium-chip-p2')).toBeDefined();
    expect(screen.getByTestId('podium-chip-p3')).toBeDefined();
  });

  it('uses team color for the chip swatch', () => {
    render(<PodiumChips entries={podium} />);
    const verChip = screen.getByTestId('podium-chip-p1');
    const swatch = within(verChip).getByTestId('podium-chip-color');
    // Red Bull color from teamColors.ts
    expect((swatch as HTMLElement).style.backgroundColor).toBeTruthy();
  });

  it('renders an em dash when entries are empty', () => {
    const { container } = render(<PodiumChips entries={[]} />);
    expect(container.textContent).toContain('—');
  });

  it('renders an em dash when entries are undefined', () => {
    const { container } = render(<PodiumChips />);
    expect(container.textContent).toContain('—');
  });
});
