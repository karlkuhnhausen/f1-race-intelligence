import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import TireCompound from "@/features/design-system/TireCompound";

describe("TireCompound", () => {
  it.each([
    ["soft", "S", "rgb(232, 0, 45)"],
    ["medium", "M", "rgb(255, 193, 7)"],
    ["hard", "H", "rgb(255, 255, 255)"],
    ["intermediate", "I", "rgb(33, 150, 243)"],
    ["wet", "W", "rgb(76, 175, 80)"],
  ])("renders compound %s with letter %s and color %s", (compound, letter, color) => {
    render(<TireCompound compound={compound} />);
    const badge = screen.getByTestId("tire-compound");
    expect(badge.textContent).toBe(letter);
    expect(badge.style.backgroundColor).toBe(color);
  });

  it("renders unknown for null compound", () => {
    render(<TireCompound compound={null} />);
    const badge = screen.getByTestId("tire-compound");
    expect(badge.textContent).toBe("?");
    expect(badge.style.backgroundColor).toBe("rgb(136, 136, 170)");
  });

  it("renders unknown for unrecognized compound string", () => {
    render(<TireCompound compound="banana" />);
    const badge = screen.getByTestId("tire-compound");
    expect(badge.textContent).toBe("?");
  });

  it("supports sm and md sizes", () => {
    const { rerender } = render(<TireCompound compound="soft" size="sm" />);
    let badge = screen.getByTestId("tire-compound");
    expect(badge.className).toContain("h-6");
    expect(badge.className).toContain("w-6");

    rerender(<TireCompound compound="soft" size="md" />);
    badge = screen.getByTestId("tire-compound");
    expect(badge.className).toContain("h-8");
    expect(badge.className).toContain("w-8");
  });
});
