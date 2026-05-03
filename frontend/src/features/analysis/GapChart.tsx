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
import type { IntervalDriver } from './analysisTypes';

interface GapChartProps {
  intervals: IntervalDriver[];
}

interface TooltipEntry {
  value?: number | null;
  name?: string;
  color?: string;
  dataKey?: string | number;
}

/** Custom tooltip that lists drivers sorted by gap to leader (smallest first). */
function GapTooltip({ active, payload, label }: { active?: boolean; payload?: TooltipEntry[]; label?: string | number }) {
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
          <span style={{ color: 'hsl(var(--muted-foreground))', minWidth: 40, textAlign: 'right' }}>
            +{(entry.value as number).toFixed(1)}s
          </span>
          <span style={{ color: entry.color }}>{entry.name}</span>
        </div>
      ))}
    </div>
  );
}

/**
 * Gap to Leader Progression — shows how time gaps evolve over the race.
 * Lapped drivers (gap = -1 sentinel) are excluded from display.
 */
export default function GapChart({ intervals }: GapChartProps) {
  // Find max laps across all drivers
  const maxLaps = Math.max(
    ...intervals.map((d) => (d.laps.length > 0 ? d.laps[d.laps.length - 1].lap : 0)),
  );

  // Transform: array of { lap, [acronym]: gap }
  const lapData: Record<string, number | null>[] = [];
  for (let lap = 1; lap <= maxLaps; lap++) {
    const point: Record<string, number | null> = { lap };
    for (const driver of intervals) {
      const entry = driver.laps.find((l) => l.lap === lap);
      if (entry && entry.gap_to_leader >= 0) {
        point[driver.driver_acronym] = entry.gap_to_leader;
      } else {
        point[driver.driver_acronym] = null; // lapped or no data
      }
    }
    lapData.push(point);
  }

  // Only show top ~10 drivers to avoid chart clutter
  const driversToShow = intervals.slice(0, 10);

  return (
    <div className="w-full rounded-lg border border-border bg-surface p-4">
      <ResponsiveContainer width="100%" height={400}>
        <LineChart
          data={lapData}
          margin={{ top: 10, right: 30, left: 10, bottom: 10 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
          <XAxis
            dataKey="lap"
            type="number"
            domain={[1, maxLaps]}
            label={{ value: 'Lap', position: 'insideBottom', offset: -5 }}
            stroke="hsl(var(--muted-foreground))"
            tick={{ fontSize: 11 }}
          />
          <YAxis
            label={{
              value: 'Gap to Leader (s)',
              angle: -90,
              position: 'insideLeft',
            }}
            stroke="hsl(var(--muted-foreground))"
            tick={{ fontSize: 11 }}
          />
          <Tooltip content={<GapTooltip />} />
          <Legend wrapperStyle={{ fontSize: '11px' }} />
          {driversToShow.map((driver) => (
            <Line
              key={driver.driver_number}
              type="monotone"
              dataKey={driver.driver_acronym}
              name={driver.driver_acronym}
              stroke={`#${driver.team_colour || '888888'}`}
              dot={false}
              strokeWidth={1.5}
              connectNulls
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
