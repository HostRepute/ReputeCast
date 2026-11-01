/**
 * The checkbox that picks every row shown, or none; mixed while some are
 * picked (review 2026-11 FE-3b). Goes with useSelection (lib/selection).
 */
export function SelectAll({ all, some, label, onChange, className }: { all: boolean; some: boolean; label: string; onChange: (on: boolean) => void; className?: string }) {
  return (
    <input
      type="checkbox"
      className={className}
      checked={all}
      ref={(el) => {
        if (el) el.indeterminate = some && !all
      }}
      aria-label={label}
      onChange={(e) => onChange(e.target.checked)}
    />
  )
}
