import { createFileRoute } from '@tanstack/react-router'
import { AccessPage } from '@/features/website/AccessPage'
import { OnlyWhenOn } from '@/features/website/WebsiteTab'

/** Forms & pages › Website › Access (WebsiteAccess). */
export const Route = createFileRoute('/w/$workspace/_app/forms/website/access')({
  component: function AccessRoute() {
    return <OnlyWhenOn>{(ov) => <AccessPage overview={ov} />}</OnlyWhenOn>
  },
})
