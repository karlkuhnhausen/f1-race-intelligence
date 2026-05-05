import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { fetchSessionAnalysis } from './analysisApi';
import type { SessionAnalysisResponse } from './analysisTypes';
import { fetchRoundDetail } from '../rounds/roundApi';
import PositionChart from './PositionChart';
import GapChart from './GapChart';
import TireStrategyChart from './TireStrategyChart';
import PitStopTimeline from './PitStopTimeline';

/**
 * Derive finishing order from position data: each driver's last lap position = final position.
 * Returns a Map<driver_number, finishing_position>.
 */
function buildDriverOrder(data: SessionAnalysisResponse): Map<number, number> {
  const order = new Map<number, number>();
  for (const driver of data.positions) {
    if (driver.laps.length > 0) {
      const lastLap = driver.laps[driver.laps.length - 1];
      order.set(driver.driver_number, lastLap.position);
    }
  }
  return order;
}

/**
 * Compute the true total laps from ALL data sources — position laps,
 * stint lap_end, and pit lap. Takes the max across these.
 * NOTE: Intervals are excluded because OpenF1 sometimes reports interval data
 * beyond actual race distance (e.g., 70 laps for a 58-lap race).
 * Stints lap_end is the most reliable indicator of actual race length.
 */
function computeTotalLaps(data: SessionAnalysisResponse): number {
  let max = data.total_laps;

  for (const driver of data.positions) {
    for (const lap of driver.laps) {
      if (lap.lap > max) max = lap.lap;
    }
  }

  if (data.stints) {
    for (const stint of data.stints) {
      if (stint.lap_end > max) max = stint.lap_end;
    }
  }

  if (data.pits) {
    for (const pit of data.pits) {
      if (pit.lap > max) max = pit.lap;
    }
  }

  return max;
}

/**
 * Build a map of driver_number → team_colour hex string from position data.
 */
function buildTeamColors(data: SessionAnalysisResponse): Map<number, string> {
  const colors = new Map<number, string>();
  for (const driver of data.positions) {
    if (driver.team_colour) {
      colors.set(driver.driver_number, driver.team_colour);
    }
  }
  return colors;
}

export default function AnalysisPage() {
  const { round, sessionType } = useParams<{
    round: string;
    sessionType: string;
  }>();
  const [data, setData] = useState<SessionAnalysisResponse | null>(null);
  const [gpName, setGpName] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [notAvailable, setNotAvailable] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!round || !sessionType) return;

    setLoading(true);
    setError(null);
    setNotAvailable(false);

    const roundNum = parseInt(round, 10);

    Promise.all([
      fetchSessionAnalysis(roundNum, sessionType),
      fetchRoundDetail(roundNum).then((r) => ({ race: r.race_name, circuit: r.circuit_name })).catch(() => null),
    ])
      .then(([result, roundInfo]) => {
        if (result === null) {
          setNotAvailable(true);
        } else {
          setData(result);
        }
        setGpName(roundInfo ? `${roundInfo.race} — ${roundInfo.circuit}` : null);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Failed to load analysis');
      })
      .finally(() => setLoading(false));
  }, [round, sessionType]);

  const sessionLabel =
    sessionType === 'sprint' ? 'Sprint' : 'Race';

  return (
    <div>
      <Link
        to={`/rounds/${round}`}
        className="inline-block font-display text-sm uppercase tracking-wider text-accent-cyan hover:text-foreground transition-colors"
      >
        ← Back to Round {round} {sessionLabel} Results
      </Link>

      <div className="mt-2 mb-6">
        <p className="font-display text-xs uppercase tracking-[0.2em] text-accent-red">
          Round {round} — {sessionLabel} Analysis
        </p>
        <h2 className="mt-1 font-display text-3xl font-bold tracking-tight">
          {gpName ?? `Round ${round}`}
        </h2>
      </div>

      {loading && (
        <p className="text-muted-foreground">Loading analysis data...</p>
      )}

      {error && (
        <p className="text-red-400">Error: {error}</p>
      )}

      {notAvailable && (
        <div className="rounded-lg border border-border bg-surface p-6 text-center">
          <p className="text-muted-foreground">
            Analysis not yet available.
          </p>
          <p className="text-sm text-muted-foreground mt-2">
            Data becomes available approximately 2 hours after session end.
          </p>
        </div>
      )}

      {data && (() => {
        const totalLaps = computeTotalLaps(data);
        const driverOrder = buildDriverOrder(data);
        const teamColors = buildTeamColors(data);
        return (
        <div className="space-y-8">
          {/* Position Battle Chart */}
          {data.positions && data.positions.length > 0 && (
            <section>
              <h3 className="font-display text-lg font-semibold mb-3">
                Position Battle
              </h3>
              <PositionChart
                positions={data.positions}
                totalLaps={totalLaps}
                overtakes={data.overtakes}
              />
            </section>
          )}

          {/* Gap to Leader Progression */}
          <section>
            <h3 className="font-display text-lg font-semibold mb-3">
              Gap to Leader
            </h3>
            {data.intervals && data.intervals.length > 0 ? (
              <GapChart intervals={data.intervals} totalLaps={totalLaps} />
            ) : (
              <p className="text-muted-foreground text-sm py-4">
                Interval data not available for this session.
              </p>
            )}
          </section>

          {/* Tire Strategy Swimlane */}
          {data.stints && data.stints.length > 0 && (
            <section>
              <h3 className="font-display text-lg font-semibold mb-3">
                Tire Strategy
              </h3>
              <TireStrategyChart
                stints={data.stints}
                totalLaps={totalLaps}
                driverOrder={driverOrder}
                teamColors={teamColors}
              />
            </section>
          )}

          {/* Pit Stop Timeline */}
          {data.pits && data.pits.length > 0 && (
            <section>
              <h3 className="font-display text-lg font-semibold mb-3">
                Pit Stops
              </h3>
              <PitStopTimeline
                pits={data.pits}
                totalLaps={totalLaps}
                driverOrder={driverOrder}
                teamColors={teamColors}
              />
            </section>
          )}
        </div>
        );
      })()}
    </div>
  );
}
