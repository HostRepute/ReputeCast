import type { ReactNode } from 'react'

/**
 * The states boards' alert (Report and Overview error states): what failed,
 * what ReputeCast did about it and what to do, on a light red box.
 */
export function StateAlert({ title, children, actions }: { title: ReactNode; children: ReactNode; actions?: ReactNode }) {
  return (
    <div role="alert" className="flex items-start gap-3.5 rounded-[10px] border border-[#FECACA] bg-[#FEF2F2] px-4 py-3.5">
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="#B91C1C" strokeWidth="1.8" aria-hidden="true" className="flex-none">
        <circle cx="10" cy="10" r="8" />
        <path d="M10 6v5M10 13.5v.5" strokeLinecap="round" />
      </svg>
      <div className="flex flex-[1_1_300px] flex-col gap-1">
        <span className="font-semibold text-[#991B1B]">{title}</span>
        <span className="text-[13px] text-[#7F1D1D]">{children}</span>
        {actions ? <div className="mt-1.5 flex flex-wrap gap-2.5">{actions}</div> : null}
      </div>
    </div>
  )
}
