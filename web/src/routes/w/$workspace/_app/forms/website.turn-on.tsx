import { createFileRoute } from '@tanstack/react-router'
import { TurnOnPage } from '@/features/website/TurnOn'
import { WithWebsite } from '@/features/website/WebsiteFrame'

/** Forms & pages › Website › Turn on the website (WebsiteStates · ready, address problems, read-only). */
export const Route = createFileRoute('/w/$workspace/_app/forms/website/turn-on')({
  component: function TurnOnRoute() {
    return <WithWebsite>{(ov) => <TurnOnPage overview={ov} />}</WithWebsite>
  },
})
