import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, render, screen } from '@testing-library/react';
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
    // 'Completed' appears both in the status badge and in the completed-at
    // line; just assert at least one appears.
    expect(screen.getAllByText('Completed').length).toBeGreaterThanOrEqual(1);
  });

  it('orders sessions newest-first when all sessions are completed', async () => {
    const { fetchRoundDetail } = await import('../../src/features/rounds/roundApi');
    (fetchRoundDetail as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      year: 2026,
      round: 1,
      race_name: 'Australian Grand Prix',
      circuit_name: 'Albert Park',
      country_name: 'Australia',
      data_as_of_utc: '2026-03-16T00:00:00Z',
      sessions: [
        {
          session_name: 'Practice 1',
          session_type: 'practice1',
          status: 'completed',
          date_start_utc: '2026-03-13T01:30:00Z',
          date_end_utc: '2026-03-13T02:30:00Z',
          results: [],
        },
        {
          session_name: 'Qualifying',
          session_type: 'qualifying',
          status: 'completed',
          date_start_utc: '2026-03-14T06:00:00Z',
          date_end_utc: '2026-03-14T07:00:00Z',
          results: [],
        },
        {
          session_name: 'Race',
          session_type: 'race',
          status: 'completed',
          date_start_utc: '2026-03-15T05:00:00Z',
          date_end_utc: '2026-03-15T07:00:00Z',
          results: [],
        },
      ],
    } as RoundDetailResponse);

    renderRoundDetail();

    // Wait for content to load.
    await screen.findByText('Race');

    // Session names render as h3 inside SessionCard.
    const headings = screen.getAllByRole('heading', { level: 3 });
    const sessionHeadings = headings
      .map((h) => h.textContent ?? '')
      .filter((t) => ['Race', 'Qualifying', 'Practice 1'].includes(t));
    expect(sessionHeadings).toEqual(['Race', 'Qualifying', 'Practice 1']);
  });

  it('preserves chronological order when not all sessions are completed', async () => {
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
          session_name: 'Race',
          session_type: 'race',
          status: 'upcoming',
          date_start_utc: '2026-05-03T13:00:00Z',
          date_end_utc: '2026-05-03T15:00:00Z',
          results: [],
        },
        {
          session_name: 'Practice 1',
          session_type: 'practice1',
          status: 'completed',
          date_start_utc: '2026-05-01T10:00:00Z',
          date_end_utc: '2026-05-01T11:00:00Z',
          results: [],
        },
        {
          session_name: 'Qualifying',
          session_type: 'qualifying',
          status: 'upcoming',
          date_start_utc: '2026-05-02T13:00:00Z',
          date_end_utc: '2026-05-02T14:00:00Z',
          results: [],
        },
      ],
    } as RoundDetailResponse);

    renderRoundDetail('8');

    await screen.findByText('Race');

    const headings = screen.getAllByRole('heading', { level: 3 });
    const sessionHeadings = headings
      .map((h) => h.textContent ?? '')
      .filter((t) => ['Race', 'Qualifying', 'Practice 1'].includes(t));
    expect(sessionHeadings).toEqual(['Practice 1', 'Qualifying', 'Race']);
  });
});

describe('RoundDetailPage — session countdowns and completed time', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    // Pin "now" so weekend-window math is deterministic.
    vi.setSystemTime(new Date('2026-05-01T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  async function mockResponse(payload: RoundDetailResponse) {
    const { fetchRoundDetail } = await import(
      '../../src/features/rounds/roundApi'
    );
    (fetchRoundDetail as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      payload,
    );
  }

  function renderAndFlush(round = '8') {
    const result = render(
      <MemoryRouter initialEntries={[`/rounds/${round}?year=2026`]}>
        <Routes>
          <Route path="/rounds/:round" element={<RoundDetailPage />} />
        </Routes>
      </MemoryRouter>,
    );
    return result;
  }

  it('renders an "Until …" countdown for upcoming sessions during the race weekend', async () => {
    await mockResponse({
      year: 2026,
      round: 8,
      race_name: 'Test GP',
      circuit_name: 'Test Circuit',
      country_name: 'Testland',
      data_as_of_utc: '2026-05-01T12:00:00Z',
      sessions: [
        {
          session_name: 'Practice 1',
          session_type: 'practice1',
          status: 'upcoming',
          // Exactly 1 hour from system time.
          date_start_utc: '2026-05-01T13:00:00Z',
          date_end_utc: '2026-05-01T14:00:00Z',
          results: [],
        },
      ],
    });

    renderAndFlush();

    // Flush the resolved fetch promise.
    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByText('Until Practice 1')).toBeDefined();
    expect(screen.getByText('1h')).toBeDefined();
    expect(screen.getByText('0m')).toBeDefined();
    expect(screen.getByText('0s')).toBeDefined();

    // Tick 1 second; seconds segment changes.
    await act(async () => {
      vi.advanceTimersByTime(1_000);
    });
    expect(screen.queryByText('0s')).toBeNull();
    expect(screen.getByText('59s')).toBeDefined();
  });

  it('renders a "live — ends in" countdown for in_progress sessions targeting end time', async () => {
    await mockResponse({
      year: 2026,
      round: 8,
      race_name: 'Test GP',
      circuit_name: 'Test Circuit',
      country_name: 'Testland',
      data_as_of_utc: '2026-05-01T12:00:00Z',
      sessions: [
        {
          session_name: 'Qualifying',
          session_type: 'qualifying',
          status: 'in_progress',
          // Started 30 minutes ago, ends in 30 minutes.
          date_start_utc: '2026-05-01T11:30:00Z',
          date_end_utc: '2026-05-01T12:30:00Z',
          results: [],
        },
      ],
    });

    renderAndFlush();

    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByText('Qualifying live — ends in')).toBeDefined();
    expect(screen.getByText('30m')).toBeDefined();
  });

  it('does NOT render a countdown for completed sessions and shows a "Completed …" line', async () => {
    await mockResponse({
      year: 2026,
      round: 1,
      race_name: 'Australian Grand Prix',
      circuit_name: 'Albert Park',
      country_name: 'Australia',
      data_as_of_utc: '2026-03-16T00:00:00Z',
      sessions: [
        {
          session_name: 'Race',
          session_type: 'race',
          status: 'completed',
          date_start_utc: '2026-03-15T05:00:00Z',
          date_end_utc: '2026-03-15T07:00:00Z',
          results: [],
        },
      ],
    });

    renderAndFlush('1');

    await act(async () => {
      await Promise.resolve();
    });

    // No countdown component.
    expect(screen.queryByTestId('race-countdown')).toBeNull();

    // "Completed " line present, with a localized end-time string after it.
    const completedLine = screen.getByTestId('session-completed-at');
    expect(completedLine).toBeDefined();
    // The localized time element exposes the raw ISO via title for
    // accessibility / power users.
    const tooltip = completedLine.querySelector('time[title]');
    expect(tooltip?.getAttribute('title')).toBe('2026-03-15T07:00:00Z');
  });

  it('does NOT render countdowns when the round is more than 7 days out', async () => {
    await mockResponse({
      year: 2026,
      round: 12,
      race_name: 'Future GP',
      circuit_name: 'Future Circuit',
      country_name: 'Futureland',
      data_as_of_utc: '2026-05-01T12:00:00Z',
      sessions: [
        {
          session_name: 'Practice 1',
          session_type: 'practice1',
          status: 'upcoming',
          // 30 days out.
          date_start_utc: '2026-05-31T10:00:00Z',
          date_end_utc: '2026-05-31T11:00:00Z',
          results: [],
        },
        {
          session_name: 'Race',
          session_type: 'race',
          status: 'upcoming',
          date_start_utc: '2026-06-02T13:00:00Z',
          date_end_utc: '2026-06-02T15:00:00Z',
          results: [],
        },
      ],
    });

    renderAndFlush('12');

    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.queryByTestId('race-countdown')).toBeNull();
    expect(screen.queryByText(/^Until /)).toBeNull();
  });

  it('handles a mixed weekend (completed + in_progress + upcoming) correctly', async () => {
    await mockResponse({
      year: 2026,
      round: 8,
      race_name: 'Mixed GP',
      circuit_name: 'Mixed Circuit',
      country_name: 'Mixedland',
      data_as_of_utc: '2026-05-01T12:00:00Z',
      sessions: [
        {
          session_name: 'Practice 1',
          session_type: 'practice1',
          status: 'completed',
          date_start_utc: '2026-04-30T10:00:00Z',
          date_end_utc: '2026-04-30T11:00:00Z',
          results: [],
        },
        {
          session_name: 'Practice 2',
          session_type: 'practice2',
          status: 'in_progress',
          date_start_utc: '2026-05-01T11:30:00Z',
          date_end_utc: '2026-05-01T12:30:00Z',
          results: [],
        },
        {
          session_name: 'Qualifying',
          session_type: 'qualifying',
          status: 'upcoming',
          date_start_utc: '2026-05-02T13:00:00Z',
          date_end_utc: '2026-05-02T14:00:00Z',
          results: [],
        },
        {
          session_name: 'Race',
          session_type: 'race',
          status: 'upcoming',
          date_start_utc: '2026-05-03T13:00:00Z',
          date_end_utc: '2026-05-03T15:00:00Z',
          results: [],
        },
      ],
    });

    renderAndFlush('8');

    await act(async () => {
      await Promise.resolve();
    });

    // Two countdowns for upcoming + one for in_progress = 3 total.
    expect(screen.getAllByTestId('race-countdown').length).toBe(3);
    expect(screen.getByText('Until Qualifying')).toBeDefined();
    expect(screen.getByText('Until Race')).toBeDefined();
    expect(screen.getByText('Practice 2 live — ends in')).toBeDefined();
    // Completed FP1 has the local-time line.
    expect(screen.getByTestId('session-completed-at')).toBeDefined();
  });

  it('cleans up countdown intervals on unmount without warnings', async () => {
    await mockResponse({
      year: 2026,
      round: 8,
      race_name: 'Test GP',
      circuit_name: 'Test Circuit',
      country_name: 'Testland',
      data_as_of_utc: '2026-05-01T12:00:00Z',
      sessions: [
        {
          session_name: 'Practice 1',
          session_type: 'practice1',
          status: 'upcoming',
          date_start_utc: '2026-05-01T13:00:00Z',
          date_end_utc: '2026-05-01T14:00:00Z',
          results: [],
        },
      ],
    });

    const { unmount } = renderAndFlush();

    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByTestId('race-countdown')).toBeDefined();

    unmount();

    // After unmount, advancing timers must not throw or cause act() warnings.
    await act(async () => {
      vi.advanceTimersByTime(5_000);
    });
    expect(screen.queryByTestId('race-countdown')).toBeNull();
  });
});
