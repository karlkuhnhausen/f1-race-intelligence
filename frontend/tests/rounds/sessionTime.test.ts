import { describe, it, expect } from 'vitest';
import {
  formatLocalDateTime,
  isWithinWeekendWindow,
} from '../../src/features/rounds/sessionTime';
import type { SessionDetail } from '../../src/features/rounds/roundApi';

function s(overrides: Partial<SessionDetail>): SessionDetail {
  return {
    session_name: 'Race',
    session_type: 'race',
    status: 'upcoming',
    date_start_utc: '2026-05-10T13:00:00Z',
    date_end_utc: '2026-05-10T15:00:00Z',
    results: [],
    ...overrides,
  };
}

describe('formatLocalDateTime', () => {
  it('returns empty string for empty input', () => {
    expect(formatLocalDateTime('')).toBe('');
  });

  it('returns empty string for invalid input', () => {
    expect(formatLocalDateTime('not-a-date')).toBe('');
  });

  it('formats a valid ISO timestamp into a human-readable string', () => {
    const out = formatLocalDateTime('2026-03-15T05:00:00Z');
    // Locale and TZ vary by runtime; assert structural pieces only.
    // Should include weekday short, month short, day, time, and TZ name.
    expect(out).not.toBe('');
    // Must contain a 3-letter weekday abbreviation followed by a comma.
    expect(out).toMatch(/^[A-Z][a-z]{2},/);
    // Must contain a 3-letter month abbreviation.
    expect(out).toMatch(/Mar/);
    // Must include a digit (day-of-month) and a colon (time).
    expect(out).toMatch(/\d/);
    expect(out).toMatch(/:/);
  });
});

describe('isWithinWeekendWindow', () => {
  const NOW = new Date('2026-05-08T12:00:00Z').getTime();

  it('returns false for empty session list', () => {
    expect(isWithinWeekendWindow([], NOW)).toBe(false);
  });

  it('returns false when all sessions are completed', () => {
    expect(
      isWithinWeekendWindow(
        [
          s({ status: 'completed', date_start_utc: '2026-04-01T00:00:00Z' }),
          s({ status: 'completed', date_start_utc: '2026-04-02T00:00:00Z' }),
        ],
        NOW,
      ),
    ).toBe(false);
  });

  it('returns true when any session is in_progress', () => {
    expect(
      isWithinWeekendWindow(
        [
          s({ status: 'completed', date_start_utc: '2026-04-01T00:00:00Z' }),
          s({ status: 'in_progress', date_start_utc: '2026-05-08T11:00:00Z' }),
          s({ status: 'upcoming', date_start_utc: '2027-01-01T00:00:00Z' }),
        ],
        NOW,
      ),
    ).toBe(true);
  });

  it('returns true when earliest upcoming starts within 7 days', () => {
    // 6 days, 23h 59m after NOW
    expect(
      isWithinWeekendWindow(
        [s({ status: 'upcoming', date_start_utc: '2026-05-15T11:59:00Z' })],
        NOW,
      ),
    ).toBe(true);
  });

  it('returns true at exactly the 7-day boundary', () => {
    expect(
      isWithinWeekendWindow(
        [s({ status: 'upcoming', date_start_utc: '2026-05-15T12:00:00Z' })],
        NOW,
      ),
    ).toBe(true);
  });

  it('returns false when earliest upcoming is just past 7 days', () => {
    expect(
      isWithinWeekendWindow(
        [s({ status: 'upcoming', date_start_utc: '2026-05-15T12:00:01Z' })],
        NOW,
      ),
    ).toBe(false);
  });

  it('respects custom windowDays', () => {
    // 2 days out, windowDays=1 → false; windowDays=3 → true
    const sessions = [
      s({ status: 'upcoming', date_start_utc: '2026-05-10T12:00:00Z' }),
    ];
    expect(isWithinWeekendWindow(sessions, NOW, 1)).toBe(false);
    expect(isWithinWeekendWindow(sessions, NOW, 3)).toBe(true);
  });
});
