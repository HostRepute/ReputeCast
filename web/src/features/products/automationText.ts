import { t } from '@/i18n'
import { ma } from './elsewhere.messages'

/*
 * The automation trigger and goal "Bought a product" in words (the canvas,
 * the palette and the step panel; features/automations/graph.ts): "any
 * product", "bought The Deliverability Field Guide".
 */

/** "Bought a product": the trigger's name. */
export function boughtTriggerName(): string {
  return t(ma.trigger)
}

/** "Bought a product": the condition's name in branches, filters and goals. */
export function boughtConditionName(): string {
  return t(ma.condition)
}

/** What the trigger points at: the product's name, "any product", or "a product" while its name isn't known. */
export function boughtRef(productId: string | undefined, name: string | undefined): string {
  if (!productId) return t(ma.anyRef)
  return name ?? t(ma.someRef)
}

/** The condition in running text: "bought The Deliverability Field Guide", or "bought a product". */
export function boughtPhrase(productId: string | undefined, name: string | undefined): string {
  return productId && name ? t(ma.phrase, { product: name }) : t(ma.phraseAny)
}

/** A goal's title: "Bought The Deliverability Field Guide", or "Bought a product". */
export function boughtGoalTitle(productId: string | undefined, name: string | undefined): string {
  return productId && name ? t(ma.goal, { product: name }) : t(ma.goalAny)
}
