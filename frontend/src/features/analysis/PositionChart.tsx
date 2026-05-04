import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
  ReferenceDot,
  Label,
} from 'recharts';
import type { PositionDriver, Overtake } from './analysisTypes';

interface PositionChartProps {
  positions: PositionDriver[];
  totalLaps: number;
  overtakes?: Overtake[];
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

  // Generate X-axis ticks at every 5-lap interval
  const xTicks: number[] = [1];
  for (let t = 5; t <= totalLaps; t += 5) {
    xTicks.push(t);
  }
  if (xTicks[xTicks.length - 1] !== totalLaps) {
    xTicks.push(totalLaps);
  }

  // Build finishing order: each driver's position on the last lap they appear in
  const finishingOrder = positions.map((driver) => {
    const lastLap = driver.laps.length > 0
      ? driver.laps.reduce((a, b) => (b.lap > a.lap ? b : a))
      : null;
    return {
      acronym: driver.driver_acronym,
      position: lastLap?.position ?? 20,
      color: `#${driver.team_colour || '888888'}`,
    };
  });

  // Build starting grid: each driver's position on lap 1
  const startingGrid = positions.map((driver) => {
    const firstLap = driver.laps.find((l) => l.lap === 1)
      ?? (driver.laps.length > 0 ? driver.laps.reduce((a, b) => (a.lap < b.lap ? a : b)) : null);
    return {
      acronym: driver.driver_acronym,
      position: firstLap?.position ?? 20,
      color: `#${driver.team_colour || '888888'}`,
    };
  });

  return (
    <div className="w-full rounded-lg border border-border bg-surface p-4">
      <ResponsiveContainer width="100%" height={500}>
        <LineChart
          data={lapData}
          margin={{ top: 10, right: 55, left: 55, bottom: 10 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="#2a2a38" />
          <XAxis
            dataKey="lap"
            type="number"
            domain={[1, totalLaps]}
            ticks={xTicks}
            label={{ value: 'Lap', position: 'insideBottom', offset: -5, fill: '#ffffff' }}
            stroke="#8888aa"
            tick={{ fontSize: 11, fill: '#ffffff' }}
          />
          <YAxis
            reversed
            domain={[1, 20]}
            stroke="#2a2a38"
            tick={false}
            axisLine={false}
          />
          <Tooltip
            cursor={{ stroke: '#555', strokeDasharray: '3 3' }}
            content={({ active, payload, label: lap }) => {
              if (!active || !payload || payload.length === 0) return null;
              const sorted = [...payload]
                .filter((e) => e.value != null)
                .sort((a, b) => (a.value as number) - (b.value as number));
              return (
                <div style={{
                  backgroundColor: 'rgba(26, 26, 35, 0.85)',
                  border: '1px solid #2a2a38',
                  borderRadius: '6px',
                  padding: '6px 10px',
                  fontSize: '11px',
                  backdropFilter: 'blur(4px)',
                  boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
                }}>
                  <p style={{ color: '#ffffff', marginBottom: 3, fontWeight: 600 }}>Lap {lap}</p>
                  {sorted.map((entry) => (
                    <div key={String(entry.dataKey)} style={{ display: 'flex', gap: 6, lineHeight: '1.6' }}>
                      <span style={{ color: '#999999', minWidth: 22, textAlign: 'right', fontWeight: 600 }}>P{entry.value}</span>
                      <span style={{ color: entry.color, fontWeight: 600 }}>{entry.name}</span>
                    </div>
                  ))}
                </div>
              );
            }}
          />
          <Legend
            wrapperStyle={{ fontSize: '11px', paddingTop: '16px' }}
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
          {finishingOrder.map((driver) => (
            <ReferenceDot
              key={`finish-${driver.acronym}`}
              x={totalLaps}
              y={driver.position}
              r={0}
            >
              <Label
                content={(props) => {
                  const vb = (props as { viewBox?: { x?: number; y?: number } }).viewBox;
                  const vx = vb?.x ?? 0;
                  const vy = vb?.y ?? 0;
                  return (
                    <text y={vy} dominantBaseline="central" fontSize={10} fontWeight={600}>
                      <tspan x={vx + 6} fill="#999999">{'P' + driver.position}</tspan>
                      <tspan dx={2} fill={driver.color}>{driver.acronym}</tspan>
                    </text>
                  );
                }}
              />
            </ReferenceDot>
          ))}
          {startingGrid.map((driver) => (
            <ReferenceDot
              key={`start-${driver.acronym}`}
              x={1}
              y={driver.position}
              r={0}
            >
              <Label
                content={(props) => {
                  const vb = (props as { viewBox?: { x?: number; y?: number } }).viewBox;
                  const vx = vb?.x ?? 0;
                  const vy = vb?.y ?? 0;
                  return (
                    <text y={vy} dominantBaseline="central" fontSize={10} fontWeight={600}>
                      <tspan x={vx - 55} fill="#999999">{'P' + driver.position}</tspan>
                      <tspan dx={2} fill={driver.color}>{driver.acronym}</tspan>
                    </text>
                  );
                }}
              />
            </ReferenceDot>
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
