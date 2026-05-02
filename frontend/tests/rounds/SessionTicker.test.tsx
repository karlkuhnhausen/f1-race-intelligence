import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, render, screen } from '@testing-library/react';
import SessionTicker from '../../src/features/rounds/SessionTicker';

describe('SessionTicker', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-01T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('countdown mode', () => {
    it('renders zero-padded HH:MM:SS to a future target', () => {
      // 1h 2m 3s in the future
      render(
        <SessionTicker
          mode="countdown"
          targetUtc="2026-05-01T13:02:03Z"
          label="Until Race"
        />,
      );
      expect(screen.getByText('01:02:03')).toBeDefined();
      expect(screen.getByText('Until Race')).toBeDefined();
      const ticker = screen.getByTestId('session-ticker');
      expect(ticker.dataset.mode).toBe('countdown');
    });

    it('rolls days into hours (unbounded HH)', () => {
      // ~3 days = 72 hours away
      render(
        <SessionTicker
          mode="countdown"
          targetUtc="2026-05-04T12:00:00Z"
          label="Until Race"
        />,
      );
      expect(screen.getByText('72:00:00')).toBeDefined();
    });

    it('decrements every second', async () => {
      render(
        <SessionTicker
          mode="countdown"
          targetUtc="2026-05-01T12:00:10Z"
          label="Until Race"
        />,
      );
      expect(screen.getByText('00:00:10')).toBeDefined();
      await act(async () => {
        vi.advanceTimersByTime(1_000);
      });
      expect(screen.getByText('00:00:09')).toBeDefined();
    });

    it('renders 00:00:00 when target is in the past', () => {
      render(
        <SessionTicker
          mode="countdown"
          targetUtc="2026-04-30T12:00:00Z"
          label="Until Race"
        />,
      );
      expect(screen.getByText('00:00:00')).toBeDefined();
    });
  });

  describe('elapsed mode', () => {
    it('renders zero-padded HH:MM:SS since start', () => {
      // 1h 30m 0s ago
      render(
        <SessionTicker
          mode="elapsed"
          targetUtc="2026-05-01T10:30:00Z"
          label="Race elapsed"
        />,
      );
      expect(screen.getByText('01:30:00')).toBeDefined();
      expect(screen.getByText('Race elapsed')).toBeDefined();
      const ticker = screen.getByTestId('session-ticker');
      expect(ticker.dataset.mode).toBe('elapsed');
    });

    it('renders 00:00:00 when start is in the future', () => {
      render(
        <SessionTicker
          mode="elapsed"
          targetUtc="2026-05-01T13:00:00Z"
          label="Race elapsed"
        />,
      );
      expect(screen.getByText('00:00:00')).toBeDefined();
    });

    it('increments every second', async () => {
      // Started 5 seconds ago.
      render(
        <SessionTicker
          mode="elapsed"
          targetUtc="2026-05-01T11:59:55Z"
          label="Race elapsed"
        />,
      );
      expect(screen.getByText('00:00:05')).toBeDefined();
      await act(async () => {
        vi.advanceTimersByTime(1_000);
      });
      expect(screen.getByText('00:00:06')).toBeDefined();
    });
  });
});
