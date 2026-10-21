import { createFileRoute } from '@tanstack/react-router'
import { GiftsPage } from '@/features/earn/gifts/GiftsPage'

/** Earn › Gifts (EarnGifts): gift options, the gift page and the gifts bought on it. */
export const Route = createFileRoute('/w/$workspace/_app/earn/gifts/')({
  component: GiftsPage,
})
