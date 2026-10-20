import { createFileRoute } from '@tanstack/react-router'
import { HomeNavPage } from '@/features/website/HomeNavPage'
import { OnlyWhenOn } from '@/features/website/WebsiteTab'

/** Forms & pages › Website › Home & navigation (WebsiteHomeNav). */
export const Route = createFileRoute('/w/$workspace/_app/forms/website/home')({
  component: function HomeNavRoute() {
    return <OnlyWhenOn>{(ov) => <HomeNavPage overview={ov} />}</OnlyWhenOn>
  },
})
