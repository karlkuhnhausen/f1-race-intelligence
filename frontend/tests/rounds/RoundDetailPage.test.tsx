import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import RoundDetailPage from '../../src/features/rounds/RoundDetailPage';
import type { RoundDetailResponse } from '../../src/features/rounds/roundApi';

const mockRoundDetail: RoundDetailResponse = {
  year: 2026,
  round: 1,
  race_name: 'Australian Grand Prix',
  circuit_name: 'Albert Park',
  country_name: 'Australia',
  data_as_of_utc: '2026-03-15T12:00:00Z',
  sessions: [
    {
      session_name: 'Race',
      session_type: 'race',
      status: 'completed',
      date_start_utc: '2026-03-15T05:00:00Z',
      date_end_utc: '2026-03-15T07:00:00Z',
      results: [
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
          driver_number: 44,
          driver_name: 'Lewis HAMILTON',
          driver_acronym: 'HAM',
          team_name: 'Ferrari',
          number_of_laps: 58,
          finishing_status: 'Finished',
          gap_to_leader: '+5.123s',
          points: 18,
        },
      ],
    },
    {
      session_name: 'Qualifying',
      session_type: 'qualifying',
      status: 'completed',
      date_start_utc: '2026-03-14T06:00:00Z',
      date_end_utc: '2026-03-14T07:00:00Z',
      results: [
        {
          position: 1,
          driver_number: 1,
          driver_name: 'Max VERSTAPPEN',
          driver_acronym: 'VER',
          team_name: 'Red Bull Racing',
          number_of_laps: 0,
          q1_time: 78.123,
          q2_time: 77.456,
          q3_time: 76.789,
        },
      ],
    },
  ],
};

vi.mock('../../src/features/rounds/roundApi', () => ({
  fetchRoundDetail: vi.fn(() => Promise.resolve(mockRoundDetail)),
}));

function renderRoundDetail(round = '1') {
  return render(
    <MemoryRouter initialEntries={[`/rounds/${round}?year=2026`]}>
      <Routes>
        <Route path="/rounds/:round" element={<RoundDetailPage />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('RoundDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders round header with race name and circuit', async () => {
    renderRoundDetail();

    expect(await screen.findByText(/Australian Grand Prix/)).toBeDefined();
    expect(screen.getByText(/Albert Park/)).toBeDefined();
    // "Australia" matches both "Australian Grand Prix" and the country text
    expect(screen.getAllByText(/Australia/).length).toBeGreaterThanOrEqual(1);
  });

  it('renders session cards for each session', async () => {
    renderRoundDetail();

    expect(await screen.findByText('Race')).toBeDefined();
    expect(screen.getByText('Qualifying')).toBeDefined();
  });

  it('displays race results with driver names', async () => {
    renderRoundDetail();

    // VER appears in both qualifying and race, HAM only in race
    expect((await screen.findAllByText('Max VERSTAPPEN')).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Lewis HAMILTON').length).toBeGreaterThanOrEqual(1);
  });

  it('shows back to calendar link', async () => {
    renderRoundDetail();

    const link = await screen.findByText(/Back to Calendar/);
    expect(link).toBeDefined();
  });

  it('renders friendly status labels for each session lifecycle state', async () => {
    const { fetchRoundDetail } = await import('../../src/features/rounds/roundApi');
    (fetchRoundDetail as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      year: 2026,
      round: 8,
      race_name: 'Test GP',
      circuit_name: 'Test Circuit',
      country_name: 'Testland',
      data_as_of_utc: '2026-04-27T12:00:00Z',
      sessions: [
        {
          session_name: 'Practice 1',
          session_type: 'practice1',
          status: 'upcoming',
          date_start_utc: '2026-05-01T10:00:00Z',
          date_end_utc: '2026-05-01T11:00:00Z',
          results: [],
        },
        {
          session_name: 'Qualifying',
          session_type: 'qualifying',
          status: 'in_progress',
          date_start_utc: '2026-04-27T10:00:00Z',
          date_end_utc: '2026-04-27T13:00:00Z',
          results: [],
        },
        {
          session_name: 'Race',
          session_type: 'race',
          status: 'completed',
          date_start_utc: '2026-04-26T10:00:00Z',
          date_end_utc: '2026-04-26T12:00:00Z',
          results: [],
        },
      ],
    } as RoundDetailResponse);

    renderRoundDetail('8');

    expect(await screen.findByText('Upcoming')).toBeDefined();
    expect(screen.getByText('Live')).toBeDefined();
    expect(screen.getByText('Completed')).toBeDefined();
  });
});
