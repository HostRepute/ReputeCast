import { useId, useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { errorMessage } from '@/api/client'
import { useWorkspace } from '@/lib/session'
import { DomainOptional, DomainRecordsCard, DomainTools, useDomainDetail } from './DomainRecords'
import { failed, useSetupActions, type StepProps } from './parts'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

/** 7.2 step 3 (SetupDomain): the provider's records for the sending domain, checked live. */
export function DomainStep({ setup, go, reason }: StepProps) {
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
    a.domain.mutate({ domain: d }, { onError: failed(`Couldn't set up ${d}`, () => choose()) })
  }
  const skip = () =>
    a.domain.mutate(
      { skip: true },
      {
        onSuccess: () => go('test'),
        onError: failed("Couldn't skip the domain step"),
      },
    )
  const name = setup.info.name
  const domainInput = (
    <form onSubmit={choose} className="contents">
      <label className="k flex items-center gap-2" htmlFor={inputId}>
        Sending domain
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
          <Button type="submit" size="sm" variant="primary" loading={a.domain.isPending} busyLabel="Fetching…" disabledReason={reason}>
            {`Use ${domain.trim().toLowerCase()}`}
          </Button>
        ) : null
      ) : (
        <Button type="submit" size="sm" variant="primary" loading={a.domain.isPending} busyLabel="Fetching…" disabledReason={reason ?? (domain.trim() ? undefined : 'Enter the domain you send from.')}>
          Show the DNS records
        </Button>
      )}
    </form>
  )
  return (
    <>
      <StepHeading title={chosen ? `Add these DNS records to ${chosen.name}` : 'Which domain will you send from?'}>
        {chosen
          ? `These are the exact records ${name} needs for this domain. Status updates live; most registrars propagate within minutes, some take up to 48 hours.`
          : `${name} signs mail for the domain in your From address. ReputeCast asks ${name} for the records that domain needs and checks them as they appear.`}
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
          <ErrorPanel title="Couldn't load the records" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        ) : (
          <section className="card flex flex-col gap-3 p-5" aria-busy="true">
            <span className="vh" role="status">
              Loading the records…
            </span>
            <Skeleton w="40%" h={16} />
            <Skeleton w="100%" h={180} />
          </section>
        )
      ) : (
        <div className="flex flex-wrap items-center gap-2.5">{domainInput}</div>
      )}
      <WizardFooter back={<Button onClick={() => go('connection')}>Back</Button>}>
        <Button onClick={skip} loading={a.domain.isPending && !chosen} disabledReason={reason}>
          Skip for now · test send still works from a verified address
        </Button>
        <Button variant="primary" className={continueClass} onClick={() => go('test')} disabledReason={chosen ? undefined : 'Choose the sending domain first, or skip for now.'}>
          Continue to test send
        </Button>
      </WizardFooter>
    </>
  )
}
