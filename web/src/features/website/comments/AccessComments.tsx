import { Link } from '@tanstack/react-router'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { andList, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import type { Access, AccessInput } from '../api'
import { ma } from './access.messages'

type Who = 'off' | 'members' | 'paid'
type Review = 'all' | 'first' | 'none'

/** Website › Access › Comments (WebsiteAccess): who can comment, which comments wait, who moderates. */
export function CommentsCard({ data, draft, set, ro }: { data: Access; draft: AccessInput; set: (p: Partial<AccessInput>) => void; ro: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const who = (draft.comments || data.comments) as Who
  const review = (draft.commentsReview || data.commentsReview) as Review
  const whoOpts: [Who, string, string][] = [
    ['off', t(ma.off), t(ma.offText)],
    ['members', t(ma.members), t(ma.membersText)],
    ['paid', t(ma.paid), t(ma.paidText)],
  ]
  const reviewOpts: [Review, string, string][] = [
    ['all', t(ma.reviewAll), t(ma.reviewAllText)],
    ['first', t(ma.reviewFirst), t(ma.reviewFirstText)],
    ['none', t(ma.reviewNone), t(ma.reviewNoneText)],
  ]
  return (
    <section className="card" id="comments" aria-labelledby="ac-comments">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ac-comments">
            {t(ma.title)}
          </h2>
          <span className="k">{t(ma.text)}</span>
        </div>
        {data.commentsWaiting > 0 ? (
          <Link to="/w/$workspace/forms/website/comments" params={{ workspace: slug }} className="lnk inline-flex items-center gap-1 font-semibold">
            {t(ma.waiting, { count: data.commentsWaiting })}
            <Icon name="arrowRight" size={14} />
          </Link>
        ) : null}
      </div>
      <div className="pad flex flex-col gap-2" role="radiogroup" aria-labelledby="ac-cwho">
        <span className="lbl" id="ac-cwho">
          {t(ma.who)}
        </span>
        {whoOpts.map(([v, title, text]) => (
          // eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its radio; the title and text inside are its text
          <label key={v} className={who === v ? 'ws-radio on' : ro ? 'ws-radio off' : 'ws-radio'}>
            <input type="radio" name="ac-cwho" checked={who === v} disabled={ro} onChange={() => set({ comments: v })} />
            <span className="ws-tline">
              <b>{title}</b>
              <span className="k2">{text}</span>
            </span>
          </label>
        ))}
      </div>
      <div className="pad flex flex-col gap-2 border-t border-line-2" role="radiogroup" aria-labelledby="ac-crev">
        <span className="lbl" id="ac-crev">
          {t(ma.review)}
        </span>
        {reviewOpts.map(([v, title, text]) => (
          // eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its radio; the title and text inside are its text
          <label key={v} className={review === v ? 'ws-radio on' : ro ? 'ws-radio off' : 'ws-radio'}>
            <input type="radio" name="ac-crev" checked={review === v} disabled={ro} onChange={() => set({ commentsReview: v })} />
            <span className="ws-tline">
              <b>{title}</b>
              <span className="k2">{text}</span>
            </span>
          </label>
        ))}
        <span className="k text-ink-2">{t(ma.links)}</span>
      </div>
      <div className="pad border-t border-line-2">
        <dl className="kv">
          <dt>{t(ma.moderatedBy)}</dt>
          <dd>
            {data.moderators.length ? andList(data.moderators) : t(ma.nobody)} <span className="k">· {t(ma.moderatedByNote)}</span>
          </dd>
          <dt>{t(ma.perPost)}</dt>
          <dd>{t(ma.perPostText)}</dd>
          <dt>{t(ma.limits)}</dt>
          <dd>{t(ma.limitsText, { chars: count(5000) })}</dd>
        </dl>
      </div>
    </section>
  )
}
