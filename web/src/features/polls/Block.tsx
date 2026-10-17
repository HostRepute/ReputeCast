import type { Component, Editor } from 'grapesjs'
import { useEffect } from 'react'
import { useT } from '@/i18n'
import { tagOf } from '@/features/composer/editor/labels'
import { ATTR_POLL, ATTR_VISIBLE } from '@/features/conditional/mjml'
import { newRuleId, ruleOf } from '@/features/conditional/rules'
import { newPollId, parsePoll } from './config'
import { applyPoll } from './Inspector'
import { m } from './messages'

/** The Blocks panel's line under the ReputeCast blocks. */
export function PollBlockHint() {
  const t = useT()
  return <p className="k blk-hint">{t(m.blockHint)}</p>
}

function ids(root: Component | undefined, skip: Component): { rules: Set<string>; polls: Set<string> } {
  const rules = new Set<string>()
  const polls = new Set<string>()
  const walk = (x: Component) => {
    if (x !== skip) {
      const r = ruleOf(x)
      if (r) rules.add(r.id)
      const raw = (x.getAttributes() as Record<string, unknown>)[ATTR_POLL]
      const p = typeof raw === 'string' ? parsePoll(raw) : null
      if (p) polls.add(p.id)
    }
    x.components().forEach(walk)
  }
  if (root) walk(root)
  return { rules, polls }
}

/**
 * A copied block gets ids of its own: a conditional block (its rule's id
 * names it in every recipient's version) and a poll (its id names its
 * answer links). The server marks a duplicate id once.
 */
export function useBlockIds(editor: Editor | null, publicUrl: string) {
  useEffect(() => {
    if (!editor) return
    const onAdd = (comp: Component) => {
      if (!tagOf(comp)) return
      const attrs = comp.getAttributes() as Record<string, unknown>
      const hasRule = typeof attrs[ATTR_VISIBLE] === 'string'
      const hasPoll = typeof attrs[ATTR_POLL] === 'string'
      if (!hasRule && !hasPoll) return
      const taken = ids(editor.getWrapper(), comp)
      const rule = ruleOf(comp)
      if (rule && taken.rules.has(rule.id)) {
        comp.addAttributes({ [ATTR_VISIBLE]: JSON.stringify({ ...rule, id: newRuleId() }) })
      }
      const poll = hasPoll ? parsePoll(String(attrs[ATTR_POLL])) : null
      if (poll && taken.polls.has(poll.id)) applyPoll(comp, { ...poll, id: newPollId() }, publicUrl)
    }
    editor.on('component:add', onAdd)
    return () => {
      editor.off('component:add', onAdd)
    }
  }, [editor, publicUrl])
}
