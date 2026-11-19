import { useNavigate } from '@tanstack/react-router'
import { Dialog as RDialog } from 'radix-ui'
import { useEffect, useId, useRef, type KeyboardEvent, type PointerEvent, type RefObject } from 'react'
import { useT } from '@/i18n'
import { useReducedMotion } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import type { PanelLayout } from './AssistantShell'
import { AsstIcon, type AsstIconName } from './icons'
import { m } from './messages'
import { closeAssistant, PANEL_MAX, PANEL_MIN, setAssistantThread, setAssistantView, setAssistantWidth, threadOf, useAssistant } from './store'
import { ThreadList } from './ThreadList'
import { ThreadComposer, ThreadLog, ThreadSub, useThreadController, type ThreadController } from './ThreadView'
import './assistant.css'

function IconButton({ icon, label, onClick, id, autoFocus }: { icon: AsstIconName; label: string; onClick: () => void; id?: string; autoFocus?: boolean }) {
  return (
    // Back to the thread takes focus when the list replaces the button that opened it.
    <button type="button" id={id} className="asst-iconbtn" aria-label={label} title={label} onClick={onClick} autoFocus={autoFocus}>
      <AsstIcon name={icon} />
    </button>
  )
}

/** The keyboard-operable separator on the docked panel's edge: 360–640 px. */
function Separator({ width }: { width: number }) {
  const t = useT()
  const drag = useRef<{ x: number; w: number } | null>(null)
  const onKeyDown = (e: KeyboardEvent) => {
    const next = e.key === 'Home' ? PANEL_MIN : e.key === 'End' ? PANEL_MAX : e.key === 'ArrowLeft' ? width + 20 : e.key === 'ArrowRight' ? width - 20 : undefined
    if (next === undefined) return
    e.preventDefault()
    setAssistantWidth(next)
  }
  const onPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    drag.current = { x: e.clientX, w: width }
    e.currentTarget.setPointerCapture(e.pointerId)
  }
  const onPointerMove = (e: PointerEvent<HTMLDivElement>) => {
    if (drag.current) setAssistantWidth(drag.current.w + (drag.current.x - e.clientX))
  }
  return (
    // A window splitter (WAI-ARIA 1.2): a focusable separator with a value.
    // eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions
    <div
      className="asst-sep"
      role="separator"
      // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
      tabIndex={0}
      aria-orientation="vertical"
      aria-label={t(m.resize)}
      aria-valuemin={PANEL_MIN}
      aria-valuemax={PANEL_MAX}
      aria-valuenow={width}
      aria-controls="assistant"
      onKeyDown={onKeyDown}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={() => (drag.current = null)}
      onPointerCancel={() => (drag.current = null)}
    />
  )
}

/** On phones the panel follows the visual viewport, so the composer stays above the keyboard. */
function useVisualViewport(ref: RefObject<HTMLElement | null>, enabled: boolean) {
  useEffect(() => {
    const vv = window.visualViewport
    const el = ref.current
    if (!enabled || !vv || !el) return
    const update = () => {
      el.style.setProperty('--asst-vv-h', `${vv.height}px`)
      el.style.setProperty('--asst-vv-top', `${vv.offsetTop}px`)
    }
    update()
    vv.addEventListener('resize', update)
    vv.addEventListener('scroll', update)
    return () => {
      vv.removeEventListener('resize', update)
      vv.removeEventListener('scroll', update)
    }
  }, [ref, enabled])
}

function Header({ c, list, onClose, dialog, titleId }: { c: ThreadController; list: boolean; onClose: () => void; dialog: boolean; titleId: string }) {
  const t = useT()
  const navigate = useNavigate()
  const title = list ? t(m.threadsTitle) : c.state.thread ? c.state.thread.title || t(m.untitled) : t(m.title)
  return (
    <div className="asst-head">
      {list ? <IconButton icon="back" label={t(m.back)} onClick={() => setAssistantView('thread')} autoFocus /> : null}
      {dialog ? (
        <RDialog.Title asChild>
          <h2>{title}</h2>
        </RDialog.Title>
      ) : (
        <h2 id={titleId}>{title}</h2>
      )}
      {list ? (
        <button
          type="button"
          className="btn"
          onClick={() => {
            c.newThread()
            setAssistantView('thread')
          }}
        >
          <AsstIcon name="plus" />
          {t(m.newThread)}
        </button>
      ) : (
        <>
          <IconButton icon="threads" label={t(m.history)} onClick={() => setAssistantView('list')} />
          {c.threadId && c.state.thread ? (
            <IconButton
              icon="expand"
              label={t(m.fullPage)}
              onClick={() => {
                const thread = c.threadId!
                closeAssistant({ refocus: false })
                void navigate({ to: '/w/$workspace/assistant/$thread', params: { workspace: c.slug, thread } })
              }}
            />
          ) : null}
        </>
      )}
      <IconButton icon="x" label={t(m.close)} onClick={onClose} />
    </div>
  )
}

function Body({ c, list, panel }: { c: ThreadController; list: boolean; panel: RefObject<HTMLElement | null> }) {
  const t = useT()
  const ui = useAssistant()
  // The composer has focus when the panel opens and when the thread view comes back.
  const { focusComposer } = c
  useEffect(() => {
    if (!list) focusComposer()
  }, [list, focusComposer])
  if (list)
    return (
      <>
        <div />
        <div className="asst-msgs">
          <ThreadList
            current={c.threadId}
            onOpen={(id) => {
              setAssistantThread(c.slug, id)
            }}
            onDeleted={(id) => {
              if (id === ui.threads[c.slug] || id === c.threadId) setAssistantThread(c.slug, undefined)
              setAssistantView('list')
            }}
          />
        </div>
        <div />
      </>
    )
  return (
    <>
      <ThreadSub c={c} />
      <ThreadLog c={c} />
      <ThreadComposer c={c} bound={() => panel.current} />
      <div className="vh" role="status" aria-live="polite">
        {c.announce || (c.state.problem ? t(m.lostTitle) : '')}
      </div>
    </>
  )
}

/**
 * The assistant panel (AssistantPanel, AssistantPanelTablet,
 * AssistantPanelPhone): docked beside the content above 1100 px (a
 * complementary landmark, resizable from 360 to 640 px), a sheet over it
 * with a scrim from 768 to 1100 px and the whole screen below, both modal
 * dialogs that keep focus inside, close on Escape and give focus back.
 */
export function AssistantPanel({ layout }: { layout: PanelLayout }) {
  const ui = useAssistant()
  const reduced = useReducedMotion()
  const panel = useRef<HTMLElement>(null)
  const titleId = useId()
  const { slug } = useWorkspace()
  void ui.threads // the thread shown changes with the store
  const c = useThreadController(threadOf(slug))
  const list = ui.view === 'list'
  const dialog = layout.mode === 'sheet' || layout.mode === 'full'
  useVisualViewport(panel, layout.mode === 'full')
  const waiting = c.state.turn?.state === 'waiting'
  const close = () => closeAssistant({ waiting })
  const aside = layout.mode === 'docked' || layout.mode === 'overlay'
  // Docked, Escape closes the panel while focus is in it; menus and dialogs
  // opened from it are portals outside it, whose Escape is their own.
  useEffect(() => {
    if (!aside) return
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key !== 'Escape' || e.defaultPrevented || !(e.target instanceof Node) || !panel.current?.contains(e.target)) return
      e.preventDefault()
      closeAssistant({ waiting })
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [aside, waiting])
  const enter = reduced ? '' : ' asst-enter'
  const content = (
    <>
      <Header c={c} list={list} onClose={close} dialog={dialog} titleId={titleId} />
      <Body c={c} list={list} panel={panel} />
    </>
  )
  if (aside) {
    return (
      <aside ref={panel} id="assistant" className={`asst asst-${layout.mode}${enter}`} aria-labelledby={titleId}>
        {layout.mode === 'docked' ? <Separator width={layout.width} /> : null}
        {content}
      </aside>
    )
  }
  return (
    <RDialog.Root
      open
      onOpenChange={(o) => {
        if (!o) close()
      }}
    >
      <RDialog.Portal>
        {/* The overlay also locks the page's scroll behind the panel; full screen it has no scrim. */}
        <RDialog.Overlay className={layout.mode === 'sheet' ? 'asst-scrim' : 'asst-scrim clear'} />
        <RDialog.Content
          ref={panel as RefObject<HTMLDivElement>}
          id="assistant"
          className={`asst asst-${layout.mode}${enter}`}
          aria-modal="true"
          aria-describedby={undefined}
          onOpenAutoFocus={(e) => e.preventDefault()}
          // closeAssistant gives focus back to what opened the panel.
          onCloseAutoFocus={(e) => e.preventDefault()}
        >
          {content}
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}
