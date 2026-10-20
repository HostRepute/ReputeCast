import type { ReactNode } from 'react'
import type { Overview } from './api'
import { WebsiteOff, TurningOn } from './TurnOn'
import { PostsPage, type PostsSearch } from './PostsPage'
import { WithWebsite } from './WebsiteFrame'
import { Page } from '@/components/shell/Page'
import { useT } from '@/i18n'
import { FormsTabs } from '@/features/forms/common'
import { m } from './messages'

/**
 * Forms & pages › Website. With the website off it holds the archive's
 * settings and "Turn on the website"; while it turns on, the progress; on,
 * the posts (Home & navigation and Access are its other views).
 */
export function WebsiteTab({ search, onSearch }: { search: PostsSearch; onSearch: (s: PostsSearch, replace?: boolean) => void }) {
  const t = useT()
  return (
    <WithWebsite>
      {(ov) =>
        ov.status === 'off' ? (
          <WebsiteOff overview={ov} />
        ) : ov.status === 'turning_on' ? (
          <Page title={t(m.screenTitle)} filters={<FormsTabs />}>
            <div className="ws-states">
              <TurningOn overview={ov} />
            </div>
          </Page>
        ) : (
          <PostsPage overview={ov} search={search} onSearch={onSearch} />
        )
      }
    </WithWebsite>
  )
}

/**
 * Home & navigation and Access exist while the website is on; off or
 * turning on, they show what the Website tab shows.
 */
export function OnlyWhenOn({ children }: { children: (ov: Overview) => ReactNode }) {
  const t = useT()
  return (
    <WithWebsite>
      {(ov) =>
        ov.status === 'on' ? (
          <>{children(ov)}</>
        ) : ov.status === 'off' ? (
          <WebsiteOff overview={ov} />
        ) : (
          <Page title={t(m.screenTitle)} filters={<FormsTabs />}>
            <div className="ws-states">
              <TurningOn overview={ov} />
            </div>
          </Page>
        )
      }
    </WithWebsite>
  )
}
