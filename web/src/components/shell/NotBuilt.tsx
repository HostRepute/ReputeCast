import { EmptyState } from '../ui/Feedback'

export function NotBuilt({ screen }: { screen: string }) {
  return <EmptyState title={`${screen} is being built`}>This screen isn't available in this build yet.</EmptyState>
}
