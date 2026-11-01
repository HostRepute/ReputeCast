import { Component, type ComponentType, type ErrorInfo, type ReactNode } from 'react'
import { t } from '@/i18n'
import { common } from '@/i18n/common'
import { Button } from './Button'
import { m } from './messages'

/**
 * Keeps one failing card, tile or chart from taking the page down (review
 * 2026-11 FE-4): it shows a small notice with Retry where the part was,
 * and the rest of the page works.
 */
export class CardBoundary extends Component<{ children: ReactNode; compact?: boolean }, { error?: Error }> {
  state: { error?: Error } = {}

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('A part of the page failed to render', error, info.componentStack)
  }

  render() {
    if (!this.state.error) return this.props.children
    return (
      <div className={this.props.compact ? 'col gap-1.5 p-3' : 'pad col gap-2'} role="alert">
        <span className="font-semibold">{t(m.partFailed)}</span>
        {this.props.compact ? null : <span className="k">{t(m.partFailedBody)}</span>}
        <span>
          <Button size="sm" onClick={() => this.setState({ error: undefined })}>
            {t(common.retry)}
          </Button>
        </span>
      </div>
    )
  }
}

/** A component wrapped in its own CardBoundary. */
export function withBoundary<P extends object>(Inner: ComponentType<P>, compact = false): ComponentType<P> {
  function Bounded(props: P) {
    return (
      <CardBoundary compact={compact}>
        <Inner {...props} />
      </CardBoundary>
    )
  }
  Bounded.displayName = `withBoundary(${Inner.displayName ?? Inner.name ?? 'Component'})`
  return Bounded
}
