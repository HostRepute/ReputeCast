import type { QueryClient } from '@tanstack/react-query'
import { ApiError, http, type Problem, type Schemas } from '@/api/client'
import type { UploadTransport } from './upload'

/** The products API (internal/products/api.go). */
export const PRODUCTS = '/api/v1/workspaces/{workspace}/products' as const
export const PRODUCT = '/api/v1/workspaces/{workspace}/products/{product}' as const
export const ORDERS = '/api/v1/workspaces/{workspace}/products/orders' as const
export const ORDER = '/api/v1/workspaces/{workspace}/products/orders/{order}' as const
export const SUMMARY = '/api/v1/workspaces/{workspace}/products/summary' as const

export type ProductDetail = Schemas['ProductDetail']
export type ProductFile = Schemas['ProductFile']
export type OrderDetail = Schemas['ProductOrderDetail']

/** Refetches every products list, order list and order after a change. */
export function invalidateProducts(qc: QueryClient) {
  void qc.invalidateQueries({ queryKey: ['get', PRODUCTS] })
  void qc.invalidateQueries({ queryKey: ['get', ORDERS] })
  void qc.invalidateQueries({ queryKey: ['get', ORDER] })
  void qc.invalidateQueries({ queryKey: ['get', SUMMARY] })
}

/** The upload calls of one product, for uploadFile(). */
export function productUploads(workspace: string, product: string, signal?: AbortSignal): UploadTransport<ProductFile> {
  const base = { workspace, product }
  return {
    async start(input) {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/products/{product}/uploads', { params: { path: base }, body: input, signal })
      return data!
    },
    chunk(id, offset, body, onSent) {
      // upload-product-chunk, sent with XMLHttpRequest: fetch can't report
      // upload progress, and one chunk is up to 4 MB.
      const path = `/api/v1/workspaces/${encodeURIComponent(workspace)}/products/${encodeURIComponent(product)}/uploads/${encodeURIComponent(id)}?offset=${offset}`
      return putBytes<Schemas['UploadState']>(path, body, signal, onSent)
    },
    async finish(id) {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/products/{product}/uploads/{upload}/complete', {
        params: { path: { ...base, upload: id } },
        signal,
      })
      return data!
    },
    async cancel(id) {
      await http.DELETE('/api/v1/workspaces/{workspace}/products/{product}/uploads/{upload}', { params: { path: { ...base, upload: id } } })
    },
  }
}

/**
 * PUTs raw bytes and reports the bytes sent as they go. Answers like the
 * API client: the JSON body, or an ApiError with the problem; a network
 * failure is a TypeError and a cancel an AbortError, as with fetch.
 */
function putBytes<T>(path: string, body: Blob, signal: AbortSignal | undefined, onSent?: (bytes: number) => void): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('PUT', path)
    xhr.setRequestHeader('Content-Type', 'application/octet-stream')
    xhr.setRequestHeader('Accept', 'application/json')
    if (onSent) xhr.upload.onprogress = (e) => onSent(e.loaded)
    xhr.onload = () => {
      let data: unknown
      try {
        data = JSON.parse(xhr.responseText)
      } catch {
        data = undefined
      }
      if (xhr.status >= 200 && xhr.status < 300) resolve(data as T)
      else reject(new ApiError(xhr.status, (data as Problem | undefined) ?? ({ status: xhr.status, title: xhr.statusText, detail: `The server answered ${xhr.status}.`, type: 'about:blank' } as Problem)))
    }
    xhr.onerror = () => reject(new TypeError('Failed to fetch'))
    xhr.onabort = () => reject(new DOMException('The upload was cancelled.', 'AbortError'))
    if (signal?.aborted) {
      reject(new DOMException('The upload was cancelled.', 'AbortError'))
      return
    }
    signal?.addEventListener('abort', () => xhr.abort(), { once: true })
    xhr.send(body)
  })
}

/** Uploads a cover image (PNG, JPEG or GIF, up to 5 MB) to the image library. */
export async function uploadCover(workspace: string, product: string, file: File): Promise<Schemas['ProductCover']> {
  const form = new FormData()
  form.append('file', file, file.name)
  const { data } = await http.POST('/api/v1/workspaces/{workspace}/products/{product}/cover', {
    params: { path: { workspace, product } },
    body: form as unknown as { file: string },
    bodySerializer: (b: unknown) => b as FormData,
  })
  return data!
}
