import { z } from 'zod'

/**
 * The contacts list's `bought` search parameter (Audience › Contacts, the
 * "Bought a product" filter): "any" for buyers of any product, or a
 * product's id. Anything else is dropped. It's ContactFilter.bought in saved
 * views, and the `bought` query parameter of list-contacts.
 */
export const boughtParam = z.union([z.literal('any'), z.uuid()]).optional().catch(undefined)
