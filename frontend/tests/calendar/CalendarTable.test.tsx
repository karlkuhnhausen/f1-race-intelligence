import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import CalendarPage from '../../src/features/calendar/CalendarPage';
import type { CalendarResponse } from '../../src/features/calendar/calendarApi';

const mockCalendar: CalendarResponse = {
  year: 2026,
  data_as_of_utc: '2026-04-19T12:00:00Z',
  next_round: 6,
  countdown_target_utc: '2026-05-10T19:00:00Z',
  rounds: [
    {
      round: 1,
      race_name: 'Australian Grand Prix',
      circuit_name: 'Albert Park',
      country_name: 'Australia',
      start_datetime_utc: '2026-03-15T05:00:00Z',
      end_datetime_utc: '2026-03-18T05:00:00Z',
      status: 'completed',
      is_cancelled: false,
    },
    {
      round: 4,
      race_name: 'Bahrain Grand Prix',
      circuit_name: 'Bahrain International Circuit',
      country_name: 'Bahrain',
      start_datetime_utc: '2026-04-05T15:00:00Z',
      end_datetime_utc: '2026-04-08T15:00:00Z',
      status: 'cancelled',
      is_cancelled: true,
      cancelled_label: 'Cancelled',
      cancelled_reason: 'Geopolitical',
    },
    {
      round: 6,
      race_name: 'Miami Grand Prix',
      circuit_name: 'Miami International Autodrome',
      country_name: 'United States',
      start_datetime_utc: '2026-05-10T19:00:00Z',
      end_datetime_utc: '2026-05-13T19:00:00Z',
      status: 'scheduled',
      is_cancelled: false,
    },
  ],
};

vi.mock('../../src/features/calendar/calendarApi', () => ({
  fetchCalendar: vi.fn(() => Promise.resolve(mockCalendar)),
}));

describe('CalendarPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the calendar table with all rounds', async () => {
    render(<MemoryRouter><CalendarPage /></MemoryRouter>);

    expect(await screen.findByText('Australian Grand Prix')).toBeDefined();
    expect(screen.getByText('Bahrain Grand Prix')).toBeDefined();
    expect(screen.getAllByText('Miami Grand Prix').length).toBeGreaterThan(0);
  });

  it('shows cancelled badge for cancelled rounds', async () => {
    render(<MemoryRouter><CalendarPage /></MemoryRouter>);

    const badge = await screen.findByText('Cancelled');
    expect(badge).toBeDefined();
    expect(badge.className).toContain('cancelled');
  });

  it('renders the next race card', async () => {
    render(<MemoryRouter><CalendarPage /></MemoryRouter>);

    expect(await screen.findByText('Next Race')).toBeDefined();
    // Miami GP appears in both the card and the table row.
    expect(screen.getAllByText('Miami Grand Prix').length).toBeGreaterThanOrEqual(2);
  });

  it('displays data freshness timestamp', async () => {
    render(<MemoryRouter><CalendarPage /></MemoryRouter>);

    const freshness = await screen.findByText(/Data as of:/);
    expect(freshness).toBeDefined();
  });
});
