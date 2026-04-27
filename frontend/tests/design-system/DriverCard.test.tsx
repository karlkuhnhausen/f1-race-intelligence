import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import DriverCard from "@/features/design-system/DriverCard";

describe("DriverCard", () => {
  it("renders name, number, position, and points", () => {
    render(
      <DriverCard
        name="Max Verstappen"
        number={1}
        constructorId="red_bull"
        position={1}
        points={425}
        gap="LEADER"
      />,
    );

    expect(screen.getByText("Max Verstappen")).toBeDefined();
    expect(screen.getByText(/#1$/)).toBeDefined();
    expect(screen.getByText(/^P1$/)).toBeDefined();
    expect(screen.getByText("425")).toBeDefined();
    expect(screen.getByText("LEADER")).toBeDefined();
  });

  it("applies the team color as a left border", () => {
    render(
      <DriverCard
        name="Lando Norris"
        number={4}
        constructorId="mclaren"
        position={2}
        points={310}
      />,
    );

    const card = screen.getByTestId("driver-card");
    // jsdom serializes hex as rgb()
    expect(card.style.borderLeft).toContain("rgb(255, 128, 0)");
  });

  it("falls back to neutral color for unknown constructors", () => {
    render(
      <DriverCard
        name="Mystery Driver"
        number={99}
        constructorId="unknown_team"
        position={20}
        points={0}
      />,
    );

    const card = screen.getByTestId("driver-card");
    // FALLBACK_TEAM_COLOR #8888aa = rgb(136, 136, 170)
    expect(card.style.borderLeft).toContain("rgb(136, 136, 170)");
  });

  it("omits the gap row when gap is not provided", () => {
    render(
      <DriverCard
        name="Charles Leclerc"
        number={16}
        constructorId="ferrari"
        position={3}
        points={280}
      />,
    );

    expect(screen.queryByText("LEADER")).toBeNull();
    // No "+" prefix anywhere when gap omitted
    expect(screen.queryByText(/^\+/)).toBeNull();
  });
});
