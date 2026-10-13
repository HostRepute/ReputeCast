import '@testing-library/jest-dom/vitest'
import './fetch'

// The API client calls `new Request('/api/v1/…')` like a browser does; Node's
// Request needs an absolute URL, so tests resolve paths against the jsdom
// page (http://localhost:3000). Mock the API with mockApi from ./api.
const NativeRequest = globalThis.Request
class PageRequest extends NativeRequest {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    super(typeof input === 'string' && input.startsWith('/') ? new URL(input, window.location.href).href : input, init)
  }
}
globalThis.Request = PageRequest as typeof Request
