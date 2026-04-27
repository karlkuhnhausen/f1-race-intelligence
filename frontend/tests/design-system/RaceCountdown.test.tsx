import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import RaceCountdown from "@/features/design-system/RaceCountdown";

describe("RaceCountdown", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-01T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders countdown digits in monospace when target is in the future", () => {
    // Target: April 22, 2026 19:00:00Z → 21d 7h 0m 0s from system time
    render(<RaceCountdown targetUtc="2026-04-22T19:00:00Z" />);
    const root = screen.getByTestId("race-countdown");
    expect(root.dataset.state).toBe("counting");
    expect(screen.getByText("21d")).toBeDefined();
    expect(screen.getByText("7h")).toBeDefined();
    expect(screen.getByText("0m")).toBeDefined();
    expect(screen.getByText("0s")).toBeDefined();
  });

  it("omits the days segment when zero", () => {
    // Target: 2 hours from system time
    render(<RaceCountdown targetUtc="2026-04-01T14:30:00Z" />);
    expect(screen.queryByText(/^0d$/)).toBeNull();
    expect(screen.getByText("2h")).toBeDefined();
    expect(screen.getByText("30m")).toBeDefined();
  });

  it("renders RACE LIVE when target is in the past", () => {
    render(<RaceCountdown targetUtc="2026-03-30T12:00:00Z" />);
    const root = screen.getByTestId("race-countdown");
    expect(root.dataset.state).toBe("expired");
    expect(screen.getByText("RACE LIVE")).toBeDefined();
  });

  it("renders the label below the digits", () => {
    render(
      <RaceCountdown
        targetUtc="2026-04-22T19:00:00Z"
        label="until lights out"
      />,
    );
    expect(screen.getByText("until lights out")).toBeDefined();
  });
});
