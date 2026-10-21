import { createFileRoute } from '@tanstack/react-router'
import { OfferEditorPage } from '@/features/earn/offers/OfferEditor'

/** An offer's editor (EarnOfferEditor). */
export const Route = createFileRoute('/w/$workspace/_app/earn/plans/offers/$offerId')({
  component: function OfferRoute() {
    const { offerId } = Route.useParams()
    return <OfferEditorPage key={offerId} offerId={offerId} />
  },
})
