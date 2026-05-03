import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { fetchSessionAnalysis } from './analysisApi';
import type { SessionAnalysisResponse } from './analysisTypes';
import PositionChart from './PositionChart';
import GapChart from './GapChart';
import TireStrategyChart from './TireStrategyChart';
import PitStopTimeline from './PitStopTimeline';

export default function AnalysisPage() {
  const { round, sessionType } = useParams<{
    round: string;
    sessionType: string;
  }>();
  const [data, setData] = useState<SessionAnalysisResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [notAvailable, setNotAvailable] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!round || !sessionType) return;

    setLoading(true);
    setError(null);
    setNotAvailable(false);

    fetchSessionAnalysis(parseInt(round, 10), sessionType)
      .then((result) => {
        if (result === null) {
          setNotAvailable(true);
        } else {
          setData(result);
        }
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
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"
      >
        ← Back to Round {round}
      </Link>

      <h2 className="font-display text-xl font-bold mb-6">
        {sessionLabel} Analysis — Round {round}
      </h2>

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

      {data && (
        <div className="space-y-8">
          {/* Position Battle Chart */}
          {data.positions && data.positions.length > 0 && (
            <section>
              <h3 className="font-display text-lg font-semibold mb-3">
                Position Battle
              </h3>
              <PositionChart
                positions={data.positions}
                totalLaps={data.total_laps}
                overtakes={data.overtakes}
              />
            </section>
          )}

          {/* Gap to Leader Progression */}
          {data.intervals && data.intervals.length > 0 && (
            <section>
              <h3 className="font-display text-lg font-semibold mb-3">
                Gap to Leader
              </h3>
              <GapChart intervals={data.intervals} />
            </section>
          )}

          {/* Tire Strategy Swimlane */}
          {data.stints && data.stints.length > 0 && (
            <section>
              <h3 className="font-display text-lg font-semibold mb-3">
                Tire Strategy
              </h3>
              <TireStrategyChart
                stints={data.stints}
                totalLaps={data.total_laps}
              />
            </section>
          )}

          {/* Pit Stop Timeline */}
          {data.pits && data.pits.length > 0 && (
            <section>
              <h3 className="font-display text-lg font-semibold mb-3">
                Pit Stops
              </h3>
              <PitStopTimeline pits={data.pits} totalLaps={data.total_laps} />
            </section>
          )}
        </div>
      )}
    </div>
  );
}
