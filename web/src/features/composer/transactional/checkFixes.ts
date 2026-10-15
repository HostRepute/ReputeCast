import type { Component } from 'grapesjs'
import { useCallback } from 'react'
import { useT } from '@/i18n'
import { useComposer } from '../context'
import { tagOf } from '../editor/labels'
import { bodyOf, componentAt, reveal } from '../fixes'
import { transactionalFooter } from './blocks'
import { m } from './messages'
import { useTx } from './state'
import type { Check, CheckFix } from './types'
import { newVariable, removeOutputs, sampleValue } from './variables'

/** Fixes the editor can apply itself; the others ("minify", a campaign setting) aren't shown. */
export const APPLICABLE = new Set<CheckFix['kind']>(['declare_variable', 'remove_variable', 'make_optional', 'add_test_value', 'insert_footer', 'set_alt', 'edit_link', 'replace_link', 'move_to_body', 'turn_off_tracking'])

/** Applies a check's fix to the editor's content, variables or test data. */
export function useApplyFix(): (check: Check, fix: CheckFix, value?: string) => void {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  return useCallback(
    (check: Check, fix: CheckFix, value?: string) => {
      const name = fix.variable ?? check.variable ?? ''
      switch (fix.kind) {
        case 'declare_variable':
          if (!name || tx.variables.some((v) => v.name === name)) return
          tx.setVariables([...tx.variables, newVariable(name)], t(m.noteDeclared, { name }))
          tx.setFocusVariable(name)
          tx.setTab('variables')
          return
        case 'remove_variable':
          if (!name) return
          // An unused declaration goes; an undeclared use is taken out of the template.
          if (check.id === 'variables.unused') tx.setVariables(tx.variables.filter((v) => v.name !== name), t(m.noteRemoved, { name }))
          else tx.removeUsages(name)
          return
        case 'make_optional':
          tx.setVariables(tx.variables.map((v) => (v.name === name ? { ...v, required: false, default: fix.value ?? v.default } : v)))
          return
        case 'add_test_value': {
          const v = tx.variables.find((x) => x.name === name)
          const set = fix.testData ?? check.testData
          if (!v || !set) return
          const val = sampleValue(v) ?? ''
          tx.setTestData(
            tx.testData.map((s) => (s.name === set ? { ...s, data: { ...s.data, [name]: val } } : s)),
            t(m.noteTestValue, { name, set }),
          )
          return
        }
        case 'insert_footer': {
          const body = bodyOf(c)
          if (!body || !c.editor) return
          const added = body.append(transactionalFooter(tx.address, tx.product, tx.kind))
          const first = (Array.isArray(added) ? added : [added]).find((x) => tagOf(x as Component))
          if (first) {
            c.editor.select(first as Component)
            ;(first as Component).getEl()?.scrollIntoView({ block: 'center', behavior: 'smooth' })
          }
          return
        }
        case 'set_alt': {
          const target = componentAt(c, fix.path ?? check.path)
          const alt = (value ?? '').trim()
          if (target && alt) target.addAttributes({ alt })
          return
        }
        case 'edit_link':
          reveal(c, fix.path ?? check.path, 'settings')
          window.setTimeout(() => document.getElementById('insp-href')?.focus(), 80)
          return
        case 'replace_link': {
          const target = componentAt(c, fix.path ?? check.path)
          if (target && fix.value) target.addAttributes({ href: fix.value })
          return
        }
        case 'move_to_body':
          if (name) c.setSubject(removeOutputs(c.subject, name).replace(/\s{2,}/g, ' ').trim())
          return
        case 'turn_off_tracking':
          tx.setMeta({ trackOpens: false, trackClicks: false })
          return
      }
    },
    [c, tx, t],
  )
}
