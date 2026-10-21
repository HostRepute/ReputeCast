import { createFileRoute } from '@tanstack/react-router'
import { OfferEditorPage } from '@/features/earn/offers/OfferEditor'

/** A new offer (EarnOfferEditor). */
export const Route = createFileRoute('/w/$workspace/_app/earn/plans/offers/new')({
  component: function NewOfferRoute() {
    return <OfferEditorPage />
  },
})
