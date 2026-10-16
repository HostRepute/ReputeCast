import { useQueryClient } from '@tanstack/react-query'
import { http, type Schemas } from '@/api/client'
import { pickableLanguages, PSEUDO_LANGUAGE, PSEUDO_LOCALE, type AppLanguage, type InstanceDefaults } from '@/i18n'
import { applyMember, type MeLike } from '@/i18n/sync'
import { fetchMe, meQueryKey, useMe } from '@/lib/session'

export type MemberLanguage = Schemas['MemberLanguage']

export const FOLLOW: MemberLanguage = { language: 'default', regionFormat: 'auto', hourCycle: 'auto', weekStart: 'auto' }

/** What the signed-in member chose, the instance defaults and the languages they can pick. */
export function useMemberLanguage() {
  const { data: me } = useMe()
  const choices: MemberLanguage = me?.user.preferences ?? FOLLOW
  const inst: InstanceDefaults = { defaultLanguage: me?.instance.defaultLanguage ?? 'en', defaultRegionFormat: me?.instance.defaultRegionFormat ?? 'standard' }
  const server = me?.instance.languages?.map((l) => l.code)
  const pseudo = !!server?.includes(PSEUDO_LOCALE)
  const languages: AppLanguage[] = pickableLanguages({ serverLanguages: server })
  // A language the member already uses stays in the list.
  if (choices.language === PSEUDO_LOCALE && !pseudo) languages.push(PSEUDO_LANGUAGE)
  return { me, choices, inst, languages, pseudo: pseudo || choices.language === PSEUDO_LOCALE }
}

/**
 * Saves the member's choices, then switches the app to them at once (the
 * screens mount again in the new language) and returns what was there
 * before, for Undo.
 */
export function useSaveMemberLanguage() {
  const qc = useQueryClient()
  return async (next: MemberLanguage) => {
    const { data } = await http.PUT('/api/v1/me/language', { body: next })
    const me = await qc.fetchQuery({ queryKey: meQueryKey, queryFn: fetchMe, staleTime: 0 })
    if (me) await applyMember(me as MeLike)
    return data
  }
}
