import type { CSSProperties } from 'react'

/** The states board's grey block (6px corners). */
function Sk({ w, h, r, style }: { w?: number | string; h: number; r?: number; style?: CSSProperties }) {
  return <span className="block rounded-md bg-[#E9ECF0]" style={{ width: w, height: h, borderRadius: r, ...style }} aria-hidden="true" />
}

/** Loading "Sender health": title, link, the ring and four lines (OverviewStates). */
export function HealthSkeleton() {
  return (
    <div className="card flex flex-col gap-4 px-[22px] py-5" aria-busy="true">
      <span className="vh" role="status">
        Loading sender health
      </span>
      <div className="flex justify-between">
        <Sk w={120} h={16} />
        <Sk w={110} h={14} />
      </div>
      <div className="flex items-center gap-5">
        <Sk w={112} h={112} r={999} style={{ flex: 'none' }} />
        <div className="flex flex-1 flex-col gap-2.5">
          <Sk w="90%" h={12} />
          <Sk w="76%" h={12} />
          <Sk w="84%" h={12} />
          <Sk w="60%" h={12} />
        </div>
      </div>
    </div>
  )
}

/** Loading "Next scheduled campaign" (OverviewStates). */
export function NextSkeleton() {
  return (
    <div className="card flex flex-col gap-4 px-[22px] py-5" aria-busy="true">
      <span className="vh" role="status">
        Loading the next campaign
      </span>
      <div className="flex justify-between">
        <Sk w={180} h={16} />
        <Sk w={140} h={22} r={999} />
      </div>
      <Sk w={220} h={22} />
      <Sk w={300} h={12} style={{ maxWidth: '100%' }} />
      <div className="grid gap-3 [grid-template-columns:repeat(4,minmax(0,1fr))]">
        <Sk h={40} />
        <Sk h={40} />
        <Sk h={40} />
        <Sk h={40} />
      </div>
      <div className="flex gap-2.5">
        <Sk w={160} h={36} r={8} />
        <Sk w={120} h={36} r={8} />
      </div>
    </div>
  )
}

/** Loading provider cards: a name, a status and three bars each (OverviewStates). */
export function ProvidersSkeleton() {
  const names: [number, number][] = [
    [110, 80],
    [90, 80],
    [150, 140],
  ]
  return (
    <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(min(340px,100%),1fr))]" aria-busy="true">
      <span className="vh" role="status">
        Loading sending providers
      </span>
      {names.map(([a, b], i) => (
        <div key={i} className="card flex flex-col gap-3 px-5 py-4.5">
          <div className="flex justify-between">
            <Sk w={a} h={16} />
            <Sk w={b} h={22} r={999} />
          </div>
          <Sk h={6} />
          <Sk h={6} />
          <Sk h={6} />
        </div>
      ))}
    </div>
  )
}

/** A generic card of grey lines for the cards below the fold. */
export function CardSkeleton({ lines }: { lines: number }) {
  return (
    <div className="card flex flex-col gap-3.5 px-[22px] py-5" aria-busy="true">
      <span className="vh" role="status">
        Loading
      </span>
      <Sk w={160} h={16} />
      {Array.from({ length: lines }, (_, i) => (
        <Sk key={i} w={`${90 - i * 9}%`} h={12} />
      ))}
    </div>
  )
}
