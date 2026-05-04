interface YearPickerProps {
  selectedYear: number;
  onYearChange: (year: number) => void;
}

export default function YearPicker({ selectedYear, onYearChange }: YearPickerProps) {
  const currentYear = new Date().getFullYear();
  const years = Array.from({ length: currentYear - 2023 + 1 }, (_, i) => currentYear - i);

  return (
    <select
      data-testid="year-picker"
      value={selectedYear}
      onChange={(e) => onYearChange(Number(e.target.value))}
      className="rounded border border-border bg-surface px-3 py-1.5 text-sm text-foreground font-mono"
    >
      {years.map((y) => (
        <option key={y} value={y}>
          {y}
        </option>
      ))}
    </select>
  );
}
