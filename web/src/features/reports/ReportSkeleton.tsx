/** Loading: tiles and a chart, shaped like the report (ReportStates). */
export function ReportSkeleton() {
  const tiles: [number, number, number][] = [
    [60, 90, 110],
    [60, 90, 70],
    [80, 90, 100],
    [90, 90, 120],
    [50, 90, 90],
    [80, 60, 50],
  ]
  return (
    <div className="flex flex-col gap-3.5" aria-busy="true">
      <span className="vh" role="status">
        Loading the report
      </span>
      <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
        {tiles.map(([a, b, c], i) => (
          <div key={i} className="card flex flex-col gap-1.5 px-4 py-3.5">
            <span className="rep-sk" style={{ width: a, height: 12 }} />
            <span className="rep-sk" style={{ width: b, height: 26 }} />
            <span className="rep-sk" style={{ width: c, height: 12 }} />
          </div>
        ))}
      </div>
      <div className="card flex flex-col gap-3 px-5 py-4.5">
        <span className="rep-sk" style={{ width: 220, height: 16 }} />
        <span className="rep-sk" style={{ height: 200 }} />
      </div>
    </div>
  )
}

