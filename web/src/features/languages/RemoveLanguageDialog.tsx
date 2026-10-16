import { useQueryClient } from '@tanstack/react-query'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { languagesKey } from './LanguagesCard'
import { md } from './messages'

type Langs = Schemas['WorkspaceLanguages']
type Lang = Schemas['WorkspaceLanguage']
type Item = Schemas['RemovalItem']

/** The id of the Default language select, for "Change the default". */
export const defaultSelectId = 'lng-default'
/** The id of the Fallback language select, for "Change the fallback". */
export const fallbackSelectId = 'lng-fallback'

/**
 * Remove a language (LanguageStates): what has a version in it and who gets
 * the fallback instead. The default and the fallback can't be removed: the
 * dialog says so and offers to change them first.
 */
export function RemoveLanguageDialog({
  language,
  data,
  open,
  onOpenChange,
}: {
  language: Lang
  data: Langs
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const qc = useQueryClient()
  const removal = $api.useQuery('get', '/api/v1/workspaces/{workspace}/languages/{code}/removal', {
    params: { path: { workspace: slug, code: language.code } },
  })
  const remove = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/languages/{code}')
  const add = $api.useMutation('post', '/api/v1/workspaces/{workspace}/languages')
  const fallbackName = data.languages.find((l) => l.code === data.fallback)?.name ?? data.fallback
  const refused = language.isDefault || language.isFallback
  const close = () => onOpenChange(false)
  const focusSelect = (id: string) => {
    close()
    window.setTimeout(() => {
      const el = document.getElementById(id)
      el?.scrollIntoView({ block: 'center' })
      el?.focus()
    }, 0)
  }
  const undo = () =>
    add.mutate(
      { params: { path: { workspace: slug } }, body: { code: language.code } },
      {
        onSuccess: (out) => qc.setQueryData(languagesKey(slug), out),
        onError: (e) => toast({ kind: 'error', title: t(md.addFailed), detail: errorMessage(e) }),
      },
    )
  const submit = () =>
    remove.mutate(
      { params: { path: { workspace: slug, code: language.code } } },
      {
        onSuccess: (out) => {
          qc.setQueryData(languagesKey(slug), out)
          toast({
            kind: 'deleted',
            title: t(md.removed, { language: language.name }),
            detail: t(md.removedDetail, { language: language.name, fallback: fallbackName }),
            undo,
          })
          close()
        },
      },
    )
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(md.eyebrow)}</span>}
      title={t(md.removeTitle, { language: language.name })}
      hideClose
      dismissible={!remove.isPending}
      footer={
        <>
          <Button onClick={close}>{t(md.cancel)}</Button>
          {refused ? (
            <Button onClick={() => focusSelect(language.isDefault ? defaultSelectId : fallbackSelectId)}>
              {language.isDefault ? t(md.changeDefault) : t(md.changeFallback)}
            </Button>
          ) : null}
          <Button
            variant="danger-primary"
            loading={remove.isPending}
            disabled={refused || removal.isPending}
            onClick={submit}
          >
            {t(md.removeButton, { language: language.name })}
          </Button>
        </>
      }
    >
      {refused ? (
        <>
          <div className="note b" role="alert">
            <Icon name="alertCircle" size={16} />
            <div style={{ flex: '1 1 320px', minWidth: 0 }}>
              {language.isDefault ? t(md.refusalDefault, { language: language.name }) : t(md.refusalFallback, { language: language.name })}
            </div>
          </div>
          {language.isDefault && language.isFallback ? (
            <span className="k" style={{ color: 'var(--ink-nav)' }}>
              {t(md.refusalNote, { language: language.name })}
            </span>
          ) : null}
        </>
      ) : removal.isPending ? (
        <div className="flex flex-col gap-2.5" aria-busy="true">
          <Skeleton h={12} />
          <Skeleton w="70%" h={12} />
          <Skeleton h={64} r={8} />
        </div>
      ) : removal.isError ? (
        <div className="note b" role="alert">
          <Icon name="alertCircle" size={16} />
          <div style={{ flex: '1 1 320px', minWidth: 0 }}>{errorMessage(removal.error)}</div>
        </div>
      ) : (
        <Affected removal={removal.data} fallback={fallbackName} />
      )}
      {remove.isError ? (
        <div className="note b" role="alert">
          <Icon name="alertCircle" size={16} />
          <div style={{ flex: '1 1 320px', minWidth: 0 }}>
            {t(md.removeFailed)} {errorMessage(remove.error)}
          </div>
        </div>
      ) : null}
    </Dialog>
  )
}

function Affected({ removal, fallback }: { removal: Schemas['LanguageRemoval']; fallback: string }) {
  const t = useT()
  const items = removal.has
  return (
    <>
      <span style={{ color: 'var(--ink-nav)' }}>
        {items.length
          ? t(md.removeText, {
              items: formatList(items.map((i) => i.name)),
              count: items.length,
              language: removal.name,
              readers: count(removal.readers),
              fallback,
            })
          : t(md.removeTextNone, { language: removal.name, readers: count(removal.readers), fallback })}
      </span>
      {items.length ? (
        <ul className="m-0 list-none p-0" style={{ border: '1px solid var(--line)', borderRadius: 8 }}>
          {items.map((i, n) => (
            <li
              key={`${i.kind}:${i.id}`}
              className="row"
              style={{ justifyContent: 'space-between', padding: '9px 12px', borderTop: n ? '1px solid var(--line-2)' : 0 }}
            >
              <b style={{ fontWeight: 600 }}>{i.name}</b>
              <span className="k">{itemDetail(i, removal.name, t)}</span>
            </li>
          ))}
        </ul>
      ) : null}
      <span className="k">{t(md.removeComeBack, { language: removal.name })}</span>
    </>
  )
}

/** "Campaign draft · German is outdated" */
function itemDetail(i: Item, language: string, t: ReturnType<typeof useT>): string {
  const kind =
    i.kind === 'campaign'
      ? i.state === 'scheduled'
        ? t(md.campaignScheduled)
        : i.state === 'waiting'
          ? t(md.campaignWaiting)
          : i.state === 'sending'
            ? t(md.campaignSending)
            : t(md.campaignDraft)
      : i.kind === 'automation_email'
        ? t(md.automationEmail)
        : i.liveVersion && i.languages?.length
          ? t(md.templateLive, { version: i.liveVersion, languages: formatList(i.languages) })
          : t(md.transactionalTemplate)
  if (!i.status) return kind
  return t(md.itemDetail, { kind, status: t(md.itemStatus, { language, status: i.status, count: i.changed }) })
}
