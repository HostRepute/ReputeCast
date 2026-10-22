import { useId, useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { errorMessage } from '@/api/client'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useWorkspace } from '@/lib/session'
import { md } from './domain.messages'
import { DomainOptional, DomainRecordsCard, DomainTools, useDomainDetail } from './DomainRecords'
import { failed, useSetupActions, type StepProps } from './parts'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

/** 7.2 step 3 (SetupDomain): the provider's records for the sending domain, checked live. */
export function DomainStep({ setup, go, reason }: StepProps) {
  const t = useT()
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const chosen = setup.domain.domainId && !setup.domain.skipped ? setup.domain : undefined
  const [domain, setDomain] = useState(chosen?.name || setup.domain.suggested)
  const inputId = useId()
  const q = useDomainDetail(slug, chosen?.domainId, setup.providerId)
  const choose = (e?: FormEvent) => {
    e?.preventDefault()
    const d = domain.trim().toLowerCase()
    if (!d || d === chosen?.name) return
    a.domain.mutate({ domain: d }, { onError: failed(t(md.setupFailed, { domain: d }), () => choose()) })
  }
  const skip = () =>
    a.domain.mutate(
      { skip: true },
      {
        onSuccess: () => go('test'),
        onError: failed(t(md.skipFailed)),
      },
    )
  const name = setup.info.name
  const domainInput = (
    <form onSubmit={choose} className="contents">
      <label className="k flex items-center gap-2" htmlFor={inputId}>
        {t(md.sendingDomain)}
        <input
          id={inputId}
          className="in mono w-[220px] max-md:w-auto max-md:min-w-0 max-md:flex-1"
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
          disabled={Boolean(reason) || a.domain.isPending}
          spellCheck={false}
          autoCapitalize="none"
        />
      </label>
      {chosen ? (
        domain.trim() && domain.trim().toLowerCase() !== chosen.name ? (
          <Button type="submit" size="sm" variant="primary" loading={a.domain.isPending} busyLabel={t(md.fetching)} disabledReason={reason}>
            {t(md.useDomain, { domain: domain.trim().toLowerCase() })}
          </Button>
        ) : null
      ) : (
        <Button type="submit" size="sm" variant="primary" loading={a.domain.isPending} busyLabel={t(md.fetching)} disabledReason={reason ?? (domain.trim() ? undefined : t(md.enterDomain))}>
          {t(md.showRecords)}
        </Button>
      )}
    </form>
  )
  return (
    <>
      <StepHeading title={chosen ? t(md.headingChosen, { domain: chosen.name }) : t(md.headingChoose)}>
        {chosen ? t(md.leadChosen, { provider: name }) : t(md.leadChoose, { provider: name })}
      </StepHeading>
      {chosen ? (
        q.data ? (
          <>
            <DomainTools workspace={slug} detail={q.data} reason={reason}>
              {domainInput}
            </DomainTools>
            <DomainRecordsCard detail={q.data} />
            <DomainOptional workspace={slug} detail={q.data} reason={reason} />
          </>
        ) : q.isError ? (
          <ErrorPanel title={t(md.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        ) : (
          <section className="card flex flex-col gap-3 p-5" aria-busy="true">
            <span className="vh" role="status">
              {t(md.loading)}
            </span>
            <Skeleton w="40%" h={16} />
            <Skeleton w="100%" h={180} />
          </section>
        )
      ) : (
        <div className="flex flex-wrap items-center gap-2.5">{domainInput}</div>
      )}
      <WizardFooter back={<Button onClick={() => go('connection')}>{t(common.back)}</Button>}>
        <Button onClick={skip} loading={a.domain.isPending && !chosen} disabledReason={reason}>
          {t(md.skip)}
        </Button>
        <Button variant="primary" className={continueClass} onClick={() => go('test')} disabledReason={chosen ? undefined : t(md.chooseFirst)}>
          {t(md.continueTest)}
        </Button>
      </WizardFooter>
    </>
  )
}
