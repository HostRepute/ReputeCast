import { EmptyState } from '../ui/Feedback'

/** Temporary scaffold for screens that are being built. */
export function NotBuilt({ screen }: { screen: string }) {
  return <EmptyState title={`${screen} is being built`}>This screen isn't available in this build yet.</EmptyState>
}
