import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import type { PositionDriver, Overtake } from './analysisTypes';

interface PositionChartProps {
  positions: PositionDriver[];
  totalLaps: number;
  overtakes?: Overtake[];
}

interface TooltipEntry {
  value?: number | null;
  name?: string;
  color?: string;
  dataKey?: string | number;
}

/** Custom tooltip that lists drivers sorted by position (P1 first). */
function PositionTooltip({ active, payload, label }: { active?: boolean; payload?: TooltipEntry[]; label?: string | number }) {
  if (!active || !payload || payload.length === 0) return null;

  const sorted = [...payload]
    .filter((entry) => entry.value != null)
    .sort((a, b) => (a.value as number) - (b.value as number));

  return (
    <div
      style={{
        backgroundColor: 'hsl(var(--surface))',
        border: '1px solid hsl(var(--border))',
        borderRadius: '6px',
        padding: '8px 12px',
        fontSize: '12px',
      }}
    >
      <p style={{ color: 'hsl(var(--foreground))', marginBottom: 4, fontWeight: 600 }}>
        Lap {label}
      </p>
      {sorted.map((entry) => (
        <div key={entry.dataKey} style={{ display: 'flex', gap: 8, lineHeight: '1.5' }}>
          <span style={{ color: 'hsl(var(--muted-foreground))', minWidth: 20, textAlign: 'right' }}>
            P{entry.value}
          </span>
          <span style={{ color: entry.color }}>{entry.name}</span>
        </div>
      ))}
    </div>
  );
}

/**
 * Position Battle Chart — lap-by-lap line chart showing all drivers.
 * Y-axis inverted (P1 at top, P20 at bottom). Each driver is a team-colored line.
 */
export default function PositionChart({
  positions,
  totalLaps,
}: PositionChartProps) {
  // Transform data for recharts: array of objects where each object
  // has { lap, [driverAcronym]: position }
  const lapData: Record<string, number>[] = [];
  for (let lap = 1; lap <= totalLaps; lap++) {
    const point: Record<string, number> = { lap };
    for (const driver of positions) {
      const lapEntry = driver.laps.find((l) => l.lap <= lap);
      // Use the last known position at or before this lap
      const closestLap = driver.laps
        .filter((l) => l.lap <= lap)
        .sort((a, b) => b.lap - a.lap)[0];
      if (closestLap) {
        point[driver.driver_acronym] = closestLap.position;
      }
    }
    lapData.push(point);
  }

  return (
    <div className="w-full rounded-lg border border-border bg-surface p-4">
      <ResponsiveContainer width="100%" height={500}>
        <LineChart
          data={lapData}
          margin={{ top: 10, right: 30, left: 10, bottom: 10 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
          <XAxis
            dataKey="lap"
            type="number"
            domain={[1, totalLaps]}
            label={{ value: 'Lap', position: 'insideBottom', offset: -5 }}
            stroke="hsl(var(--muted-foreground))"
            tick={{ fontSize: 11 }}
          />
          <YAxis
            reversed
            domain={[1, 20]}
            label={{
              value: 'Position',
              angle: -90,
              position: 'insideLeft',
            }}
            stroke="hsl(var(--muted-foreground))"
            tick={{ fontSize: 11 }}
          />
          <Tooltip content={<PositionTooltip />} />
          <Legend
            wrapperStyle={{ fontSize: '11px' }}
          />
          {positions.map((driver) => (
            <Line
              key={driver.driver_number}
              type="stepAfter"
              dataKey={driver.driver_acronym}
              name={driver.driver_acronym}
              stroke={`#${driver.team_colour || '888888'}`}
              dot={false}
              strokeWidth={1.5}
              connectNulls={false}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
