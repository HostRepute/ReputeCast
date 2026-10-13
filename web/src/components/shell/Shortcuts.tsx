import { useNavigate } from '@tanstack/react-router'
import { Dialog as RDialog } from 'radix-ui'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { focusPageSearch, GO_KEYS, modKeyLabel, runShortcutHandler, SEQUENCE_MS, shortcutFor, type GoTarget } from '@/lib/shortcuts'
import { toast } from '../ui/Toast'

const GO_PATHS: Record<GoTarget, string> = {
  overview: '/w/$workspace',
  campaigns: '/w/$workspace/campaigns',
  automations: '/w/$workspace/automations',
  audience: '/w/$workspace/audience',
  forms: '/w/$workspace/forms',
  deliverability: '/w/$workspace/deliverability',
  providers: '/w/$workspace/providers',
  settings: '/w/$workspace/settings',
}

function Keys({ keys, then }: { keys: string[]; then?: string[] }) {
  const k = (list: string[]) =>
    list.map((key, i) => (
      <span key={key}>
        {i > 0 ? ' ' : null}
        <kbd>{key}</kbd>
      </span>
    ))
  return (
    <>
      {k(keys)}
      {then ? <> then {k(then)}</> : null}{' '}
    </>
  )
}

function Row({ children }: { children: ReactNode }) {
  return <li>{children}</li>
}

/**
 * The "?" sheet (Feedback board): the board's card as a modal. Esc or a
 * click outside closes it; it opens from the keyboard only, so it needs no
 * button of its own.
 */
export function ShortcutsDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const mod = modKeyLabel()
  return (
    <RDialog.Root open={open} onOpenChange={onOpenChange}>
      <RDialog.Portal>
        <RDialog.Overlay className="ovl">
          <RDialog.Content className="dlg kdlg">
            <div className="kdh">
              <RDialog.Title className="kdt">Keyboard shortcuts</RDialog.Title>
              <span className="k" aria-hidden="true">
                press <kbd>?</kbd>
              </span>
            </div>
            <RDialog.Description className="vh">Press ? to open this sheet from anywhere, Escape to close it.</RDialog.Description>
            <ul className="kbgrid">
              <Row>
                <Keys keys={[mod, 'K']} />
                Search everything
              </Row>
              <Row>
                <Keys keys={['N']} />
                New campaign
              </Row>
              <Row>
                <Keys keys={[mod, 'S']} />
                Save version
              </Row>
              <Row>
                <Keys keys={[mod, 'Enter']} />
                Open Send Readiness
              </Row>
              <Row>
                <Keys keys={['G']} then={['O']} />
                Overview
              </Row>
              <Row>
                <Keys keys={['G']} then={['A']} />
                Audience
              </Row>
              <Row>
                <Keys keys={['Esc']} />
                Close panel or dialog
              </Row>
              <Row>
                <Keys keys={['?']} />
                This sheet
              </Row>
              {Object.entries(GO_KEYS)
                .filter(([key]) => key !== 'o' && key !== 'a')
                .map(([key, g]) => (
                  <Row key={key}>
                    <Keys keys={['G']} then={[key.toUpperCase()]} />
                    {g.label}
                  </Row>
                ))}
            </ul>
            <span className="k">Every shortcut has a visible button equivalent; none is required.</span>
          </RDialog.Content>
        </RDialog.Overlay>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

/**
 * The app-wide keyboard shortcuts. Typing in a field never triggers a
 * single-key shortcut; ⌘K, ⌘S and ⌘Enter work anywhere and only take over
 * the browser's own shortcut when the page handles them.
 */
export function GlobalShortcuts() {
  const navigate = useNavigate()
  const { slug } = useWorkspace()
  const canCreate = useCan('campaigns:edit')
  const [help, setHelp] = useState(false)
  const pending = useRef(false)
  const timer = useRef<number | undefined>(undefined)

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.defaultPrevented || e.isComposing) return
      const r = shortcutFor(e, pending.current)
      window.clearTimeout(timer.current)
      pending.current = r.pendingG
      if (r.pendingG) timer.current = window.setTimeout(() => (pending.current = false), SEQUENCE_MS)
      const a = r.action
      if (!a) return
      switch (a.type) {
        case 'search':
          if (runShortcutHandler('search') || focusPageSearch()) e.preventDefault()
          return
        case 'save':
          if (runShortcutHandler('save')) e.preventDefault()
          return
        case 'readiness':
          if (runShortcutHandler('readiness')) e.preventDefault()
          return
        case 'help':
          e.preventDefault()
          setHelp(true)
          return
        case 'new-campaign':
          e.preventDefault()
          if (canCreate) void navigate({ to: '/w/$workspace/campaigns/new', params: { workspace: slug } })
          else toast({ kind: 'info', title: deniedReason('campaigns:edit') })
          return
        case 'go':
          e.preventDefault()
          void navigate({ to: GO_PATHS[a.to], params: { workspace: slug } })
      }
    }
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.clearTimeout(timer.current)
    }
  }, [navigate, slug, canCreate])

  return <ShortcutsDialog open={help} onOpenChange={setHelp} />
}
