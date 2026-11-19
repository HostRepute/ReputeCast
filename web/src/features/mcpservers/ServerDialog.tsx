import { useState, type FormEvent } from 'react'
import { errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select, Toggle } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { roleName } from '@/lib/roles'
import { useWorkspace } from '@/lib/session'
import { tags } from '@/features/ai/parts'
import { createServer, deleteServer, startSignIn, syncServer, updateServer, useRefreshMCP, type MCPServer } from './api'
import { m } from './messages'

type Auth = MCPServer['auth']
const MANAGERS = ['owner', 'admin']

/** "Owner and Admin": a server's roles by name. */
export function rolesList(roles: string[]): string {
  // A built-in role by its translated name; a custom one by its key.
  return formatList(roles.map((key) => roleName({ key, name: '' }) || key))
}

function whoOf(roles: string[]): 'all' | 'managers' | 'custom' {
  if (!roles.length) return 'all'
  return roles.length === MANAGERS.length && MANAGERS.every((r) => roles.includes(r)) ? 'managers' : 'custom'
}

function validUrl(s: string): boolean {
  try {
    const u = new URL(s.trim())
    return u.protocol === 'https:' || u.protocol === 'http:'
  } catch {
    return false
  }
}

/**
 * Add an MCP server (MCPServers › Add) or edit one: its name, address,
 * sign-in and who can use it. Secrets are write-only: a stored one is
 * "Set", and Replace takes a new one. Adding a server without OAuth syncs
 * its tools at once; an OAuth one waits for the manager's own sign-in.
 */
export function ServerDialog({ server, onClose, onSaved }: { server?: MCPServer; onClose: () => void; onSaved: (id: string | undefined) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshMCP()
  const editing = Boolean(server)
  const [name, setName] = useState(server?.name ?? '')
  const [url, setUrl] = useState(server?.url ?? '')
  const [auth, setAuth] = useState<Auth>(server?.auth ?? 'oauth')
  const [headerName, setHeaderName] = useState(server?.headerName || 'Authorization')
  const [headerValue, setHeaderValue] = useState('')
  const [replacing, setReplacing] = useState(!server?.headerSet)
  const [clientId, setClientId] = useState(server?.clientId ?? '')
  const [clientSecret, setClientSecret] = useState('')
  const [replacingSecret, setReplacingSecret] = useState(!server?.clientSecretSet)
  const initialWho = whoOf(server?.roles ?? [])
  const [who, setWho] = useState(initialWho)
  const [enabled, setEnabled] = useState(server?.enabled ?? true)
  const [errors, setErrors] = useState<{ name?: string; url?: string; headerName?: string; headerValue?: string }>({})
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState<string>()
  const [removing, setRemoving] = useState(false)

  const roles = who === 'all' ? [] : who === 'managers' ? MANAGERS : (server?.roles ?? [])

  async function submit(e: FormEvent) {
    e.preventDefault()
    const next: typeof errors = {}
    if (!name.trim()) next.name = t(m.nameRequired)
    if (!editing && !validUrl(url)) next.url = t(m.urlInvalid)
    if (auth === 'header' && !editing && !headerName.trim()) next.headerName = t(m.headerNameRequired)
    if (auth === 'header' && replacing && !headerValue.trim()) next.headerValue = t(m.headerValueRequired)
    setErrors(next)
    if (Object.keys(next).length) {
      window.setTimeout(() => document.querySelector<HTMLElement>('.mcp-dlg [aria-invalid="true"]')?.focus())
      return
    }
    setBusy(true)
    setFailed(undefined)
    try {
      if (server) {
        const body: Schemas['RemoteMCPServerUpdate'] = { name: name.trim(), roles, enabled }
        if (server.auth === 'header' && replacing && headerValue) body.headerValue = headerValue
        if (server.auth === 'oauth') {
          if (clientId.trim() !== (server.clientId ?? '')) body.clientId = clientId.trim()
          if (replacingSecret && clientSecret) body.clientSecret = clientSecret
        }
        const saved = await updateServer(slug, server.id, body)
        await refresh()
        toast({ kind: 'success', title: t(m.saved, { server: saved.name }) })
        onSaved(saved.id)
        return
      }
      const body: Schemas['RemoteMCPServerInput'] = { name: name.trim(), url: url.trim(), auth, roles }
      if (auth === 'header') {
        body.headerName = headerName.trim()
        body.headerValue = headerValue
      }
      if (auth === 'oauth' && clientId.trim()) {
        body.clientId = clientId.trim()
        if (clientSecret) body.clientSecret = clientSecret
      }
      const created = await createServer(slug, body)
      if (auth === 'oauth') {
        // The tools are listed as the manager: their own sign-in first; back here, the page syncs.
        try {
          await startSignIn(slug, created.id)
          return
        } catch {
          toast({ kind: 'success', title: t(m.addedSignIn, { server: created.name }) })
        }
      } else {
        try {
          const synced = await syncServer(slug, created.id)
          toast({ kind: 'success', title: t(m.addedSynced, { server: created.name, count: synced.tools.filter((x) => !x.removed).length }) })
        } catch (err) {
          toast({ kind: 'error', title: t(m.addedSyncFailed, { server: created.name }), detail: errorMessage(err) })
        }
      }
      await refresh()
      onSaved(created.id)
    } catch (err) {
      setFailed(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    if (!server) return
    await deleteServer(slug, server.id)
    await refresh()
    toast({ kind: 'deleted', title: t(m.removed, { server: server.name }) })
    onSaved(undefined)
  }

  const formId = 'mcp-server-form'
  return (
    <>
      <Dialog
        open={!removing}
        onOpenChange={(o) => !o && !busy && onClose()}
        title={server ? t(m.editTitle, { server: server.name }) : t(m.addTitle)}
        footer={
          <>
            {server ? (
              <Button variant="danger" className="mr-auto" onClick={() => setRemoving(true)} disabled={busy}>
                {t(m.removeServer)}
              </Button>
            ) : null}
            <Button onClick={onClose} disabled={busy}>
              {t(common.cancel)}
            </Button>
            <Button variant="primary" type="submit" form={formId} loading={busy}>
              {server ? t(common.save) : t(m.addAndSync)}
            </Button>
          </>
        }
      >
        <form id={formId} className="mcp-dlg" onSubmit={(e) => void submit(e)} noValidate>
          {failed ? <Note tone="bad">{failed}</Note> : null}
          <Field label={t(m.fieldName)} error={errors.name}>
            <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={80} autoComplete="off" />
          </Field>
          {server ? (
            <div className="lbl">
              <span className="font-semibold">{t(m.fieldUrl)}</span>
              <span className="mono text-[12.5px] [overflow-wrap:anywhere]">{server.url}</span>
            </div>
          ) : (
            <Field label={t(m.fieldUrl)} hint={t(m.fieldUrlHint)} error={errors.url}>
              <Input value={url} onChange={(e) => setUrl(e.target.value)} inputMode="url" type="url" autoComplete="off" spellCheck={false} />
            </Field>
          )}
          {server ? null : (
            <Field label={t(m.fieldAuth)}>
              <Select value={auth} onChange={(e) => setAuth(e.target.value as Auth)}>
                <option value="oauth">{t(m.optOAuth)}</option>
                <option value="header">{t(m.optHeader)}</option>
                <option value="none">{t(m.optNone)}</option>
              </Select>
            </Field>
          )}
          {auth === 'header' ? (
            <>
              {server ? (
                <div className="lbl">
                  <span className="font-semibold">{t(m.fieldHeaderName)}</span>
                  <span className="mono text-[12.5px]">{server.headerName}</span>
                </div>
              ) : (
                <Field label={t(m.fieldHeaderName)} error={errors.headerName}>
                  <Input value={headerName} onChange={(e) => setHeaderName(e.target.value)} autoComplete="off" spellCheck={false} />
                </Field>
              )}
              {replacing ? (
                <Field label={t(m.fieldHeaderValue)} hint={t(m.secretHint)} error={errors.headerValue}>
                  <Input type="password" value={headerValue} onChange={(e) => setHeaderValue(e.target.value)} autoComplete="new-password" spellCheck={false} />
                </Field>
              ) : (
                <SecretSet label={t(m.fieldHeaderValue)} onReplace={() => setReplacing(true)} />
              )}
              {server?.headerSet && replacing ? (
                <Button
                  size="sm"
                  className="self-start"
                  onClick={() => {
                    setReplacing(false)
                    setHeaderValue('')
                  }}
                >
                  {t(m.keepCurrent)}
                </Button>
              ) : null}
            </>
          ) : null}
          {auth === 'oauth' ? (
            <details className="mcp-client" open={Boolean(server?.clientId)}>
              <summary>{t(m.clientSummary)}</summary>
              <div className="mcp-client-body">
                <Field label={t(m.fieldClientId)} hint={t(m.clientHint)} optional>
                  <Input value={clientId} onChange={(e) => setClientId(e.target.value)} autoComplete="off" spellCheck={false} />
                </Field>
                {replacingSecret ? (
                  <Field label={t(m.fieldClientSecret)} hint={t(m.secretHint)} optional>
                    <Input type="password" value={clientSecret} onChange={(e) => setClientSecret(e.target.value)} autoComplete="new-password" spellCheck={false} />
                  </Field>
                ) : (
                  <SecretSet label={t(m.fieldClientSecret)} onReplace={() => setReplacingSecret(true)} />
                )}
                {server?.redirectUri ? <small className="k [overflow-wrap:anywhere]">{t(m.redirectHint, { uri: server.redirectUri })}</small> : null}
              </div>
            </details>
          ) : null}
          <Field label={t(m.fieldWho)}>
            <Select value={who} onChange={(e) => setWho(e.target.value as typeof who)}>
              <option value="all">{t(m.whoAll)}</option>
              <option value="managers">{t(m.whoManagers)}</option>
              {initialWho === 'custom' ? <option value="custom">{t(m.whoCustom, { roles: rolesList(server?.roles ?? []) })}</option> : null}
            </Select>
          </Field>
          {server ? <Toggle label={t(m.enabled)} description={t(m.enabledHint)} checked={enabled} onChange={(e) => setEnabled(e.target.checked)} /> : null}
          <small className="k">
            <T msg={m.privateNote} values={tags} />
          </small>
        </form>
      </Dialog>
      {server ? (
        <ConfirmDialog
          open={removing}
          onOpenChange={(o) => !o && setRemoving(false)}
          badge={t(m.removeBadge)}
          title={t(m.removeTitle, { server: server.name })}
          description={t(m.removeBody)}
          confirmLabel={t(m.removeServer)}
          onConfirm={remove}
        />
      ) : null}
    </>
  )
}

/** A stored secret: never shown, only that it is set, and Replace. */
function SecretSet({ label, onReplace }: { label: string; onReplace: () => void }) {
  const t = useT()
  return (
    <div className="lbl">
      <span className="font-semibold">{label}</span>
      <span className="flex flex-wrap items-center gap-2">
        <span className="ai-mask">
          <span aria-hidden="true">••••••</span>
          <span>{t(m.secretSet)}</span>
        </span>
        <Button size="sm" onClick={onReplace}>
          {t(m.replace)}
        </Button>
      </span>
    </div>
  )
}
