import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import LapTimeDisplay from "@/features/design-system/LapTimeDisplay";

describe("LapTimeDisplay", () => {
  it("renders the time formatted as M:SS.mmm", () => {
    render(<LapTimeDisplay time={87.654} />);
    expect(screen.getByText("1:27.654")).toBeDefined();
  });

  it("renders sub-minute time without leading minutes", () => {
    render(<LapTimeDisplay time={42.123} />);
    expect(screen.getByText("42.123")).toBeDefined();
  });

  it("colors a positive delta with the positive class", () => {
    render(<LapTimeDisplay time={90.0} delta={0.123} />);
    const delta = screen.getByTestId("lap-time-delta");
    expect(delta.textContent).toBe("+0.123");
    expect(delta.className).toContain("text-positive");
  });

  it("colors a negative delta with the negative class", () => {
    render(<LapTimeDisplay time={90.0} delta={-0.456} />);
    const delta = screen.getByTestId("lap-time-delta");
    expect(delta.textContent).toBe("−0.456");
    expect(delta.className).toContain("text-negative");
  });

  it("uses neutral foreground color when delta is zero", () => {
    render(<LapTimeDisplay time={90.0} delta={0} />);
    const delta = screen.getByTestId("lap-time-delta");
    expect(delta.textContent).toBe("0.000");
    expect(delta.className).toContain("text-foreground");
  });

  it("renders only the delta when deltaOnly is true", () => {
    render(<LapTimeDisplay delta={0.789} deltaOnly />);
    expect(screen.queryByTestId("lap-time-display")).toBeNull();
    const delta = screen.getByTestId("lap-time-delta");
    expect(delta.textContent).toBe("+0.789");
  });

  it("renders no delta when delta is undefined", () => {
    render(<LapTimeDisplay time={90.0} />);
    expect(screen.queryByTestId("lap-time-delta")).toBeNull();
  });
});
