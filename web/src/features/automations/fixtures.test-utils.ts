import type { FlowGraph, NodeData, Refs, StepType } from './graph'
import type { Box } from './graph'

/*
 * The Welcome series v4 draft exactly as the AutomationFlow board draws it:
 * positions, step settings, email names and the step heights (from the
 * board's edge coordinates).
 */

const email = (emailId: string, subject: string): NodeData => ({
  email: { emailId, emailVersion: 1, subject, sendWindow: { enabled: true, start: '07:00', end: '21:00' }, skipRecentHours: 4 },
})

const node = (id: string, type: StepType, x: number, y: number, data: NodeData) => ({ id, type, position: { x, y }, data })
const edge = (source: string, target: string, path?: string) => ({ id: `${source}-${target}`, source, target, data: path ? { path } : {} })

export function welcomeV4(): FlowGraph {
  return {
    nodes: [
      node('n1', 'trigger', 370, 24, { trigger: { kind: 'joined_list', listId: 'digest', confirmedOnly: true, reentry: 'once' } }),
      node('n2', 'delay', 370, 144, { delay: { amount: 10, unit: 'minutes' } }),
      node('n3', 'email', 370, 215, email('w1', "You're in. Here's what to expect from Acme")),
      node('n4', 'branch', 370, 399, { condition: { kind: 'opened', step: 'n3', within: { amount: 2, unit: 'days' }, humanOnly: true } }),
      node('n5', 'delay', 150, 531, { delay: { amount: 2, unit: 'days' } }),
      node('n6', 'email', 150, 623, email('w2', 'Three things to try first')),
      node('n7', 'split', 150, 719, { variants: [{ key: 'A', weight: 50 }, { key: 'B', weight: 50 }] }),
      node('n8', 'email', 30, 816, email('w3a', 'Set up your first flag in 3 minutes')),
      node('n9', 'email', 270, 816, email('w3b', 'A 5-minute walkthrough')),
      node('n10', 'goal', 150, 932, { label: 'Clicked docs', condition: { kind: 'clicked', urlContains: 'docs' } }),
      node('n11', 'delay', 590, 531, { delay: { amount: 3, unit: 'days' } }),
      node('n12', 'email', 590, 623, email('wpt', 'Did you get our welcome?')),
      node('n13', 'exit', 590, 735, {}),
    ],
    edges: [
      edge('n1', 'n2'),
      edge('n2', 'n3'),
      edge('n3', 'n4'),
      edge('n4', 'n5', 'yes'),
      edge('n4', 'n11', 'no'),
      edge('n5', 'n6'),
      edge('n6', 'n7'),
      edge('n7', 'n8', 'A'),
      edge('n7', 'n9', 'B'),
      edge('n8', 'n10'),
      edge('n9', 'n10'),
      edge('n11', 'n12'),
      edge('n12', 'n13'),
    ],
  }
}

export const welcomeRefs: Refs = {
  lists: new Map([['digest', 'Weekly Digest']]),
  forms: new Map(),
  emails: new Map([
    ['w1', { name: 'Welcome · step 1 "You\'re in"', latestVersion: 5, subject: '' }],
    ['w2', { name: 'Step 2 "Three things to try first"', latestVersion: 2, subject: '' }],
    ['w3a', { name: 'Step 3 "Set up your first flag"', latestVersion: 1, subject: '' }],
    ['w3b', { name: 'Step 3 "A 5-minute walkthrough"', latestVersion: 1, subject: '' }],
    ['wpt', { name: '"Did you get our welcome?" · plain text', latestVersion: 1, subject: '' }],
  ]),
  providers: new Map(),
  fields: new Map(),
}

/** Step heights on the board (top of the next edge minus the step's top). */
export const boardHeights: Record<string, number> = {
  n1: 102,
  n2: 53,
  n3: 166,
  n4: 78,
  n5: 74,
  n6: 78,
  n7: 53,
  n8: 78,
  n9: 78,
  n10: 74,
  n11: 74,
  n12: 94,
  n13: 74,
}

export function boardBoxes(g: FlowGraph): Map<string, Box> {
  return new Map(g.nodes.map((n) => [n.id, { x: n.position.x, y: n.position.y, w: 224, h: boardHeights[n.id] ?? 60 }]))
}
