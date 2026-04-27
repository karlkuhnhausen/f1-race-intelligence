import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import StandingsTable, {
  type StandingsRow,
} from "@/features/design-system/StandingsTable";

const rows: StandingsRow[] = [
  { position: 1, name: "Max Verstappen", constructorId: "red_bull", points: 425, wins: 12 },
  { position: 2, name: "Lando Norris", constructorId: "mclaren", points: 310, wins: 4 },
  { position: 3, name: "Charles Leclerc", constructorId: "ferrari", points: 280, wins: 3 },
];

describe("StandingsTable", () => {
  it("renders the title and all rows", () => {
    render(<StandingsTable title="Drivers Championship" rows={rows} />);

    expect(screen.getByText("Drivers Championship")).toBeDefined();
    expect(screen.getByText("Max Verstappen")).toBeDefined();
    expect(screen.getByText("Lando Norris")).toBeDefined();
    expect(screen.getByText("Charles Leclerc")).toBeDefined();
    expect(screen.getAllByTestId("standings-row")).toHaveLength(3);
  });

  it("applies team color accent on the left border of each row", () => {
    render(<StandingsTable title="Drivers" rows={rows} />);
    const renderedRows = screen.getAllByTestId("standings-row");

    // jsdom serializes hex colors as rgb() in element.style
    expect(renderedRows[0].style.borderLeft).toContain("rgb(54, 113, 198)"); // red_bull
    expect(renderedRows[1].style.borderLeft).toContain("rgb(255, 128, 0)");  // mclaren
    expect(renderedRows[2].style.borderLeft).toContain("rgb(232, 0, 45)");   // ferrari
  });

  it("alternates row backgrounds", () => {
    render(<StandingsTable title="Drivers" rows={rows} />);
    const renderedRows = screen.getAllByTestId("standings-row");

    expect(renderedRows[0].className).toContain("bg-surface");
    expect(renderedRows[1].className).toContain("bg-background");
  });

  it("hides the wins column by default and shows it when requested", () => {
    const { rerender } = render(
      <StandingsTable title="Constructors" rows={rows} />,
    );
    expect(screen.queryByText("Wins")).toBeNull();

    rerender(
      <StandingsTable title="Drivers" rows={rows} columns={["wins"]} />,
    );
    expect(screen.getByText("Wins")).toBeDefined();
    expect(screen.getByText("12")).toBeDefined();
  });

  it("renders an empty state when rows are empty", () => {
    render(<StandingsTable title="Drivers" rows={[]} />);
    expect(screen.getByText(/No standings available/)).toBeDefined();
  });
});
