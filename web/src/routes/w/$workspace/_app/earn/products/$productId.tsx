import { createFileRoute } from '@tanstack/react-router'
import { ProductEditorPage } from '@/features/products/ProductEditor'

/** The product editor (ProductEditor): /earn/products/{id}. "New product" starts a draft and opens it here. */
export const Route = createFileRoute('/w/$workspace/_app/earn/products/$productId')({
  component: function ProductRoute() {
    const { productId } = Route.useParams()
    return <ProductEditorPage key={productId} productId={productId} />
  },
})
