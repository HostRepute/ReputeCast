import { defineMessages } from '@/i18n'

/**
 * The composer's Discount code block (M59 G-22, ComposerTimedBlocks): its
 * palette entry, canvas, inspector and the token menu's group. Pool names
 * and Liquid stay as written.
 */
export const mc = defineMessages('coupons.composer', {
  block: 'Discount code',
  name: 'Discount code',
  defaultLabel: 'Your code',
  defaultButton: 'Shop now',
  canvasPick: 'Pick a discount code pool in the block’s settings.',
  canvasUnknown: 'There’s no discount code pool named {slug}. Pick another in the block’s settings.',
  oneCode: 'One code per reader, given when the email is sent.',
  pool: 'Pool',
  poolOption: '{name} · {free} free',
  poolNone: 'Pick a pool',
  poolStore: '{platform} · {store}',
  poolUpload: 'Uploaded codes',
  poolOnePer: 'one code per contact',
  poolLiquid: 'in Liquid <mono>{token}</mono>',
  noPools: 'No pool yet. <a>Create one</a> in Campaigns › Discount codes.',
  label: 'Label',
  button: 'Button',
  buttonOn: 'Show the button',
  buttonOpens: 'Opens {store} with the code applied: <mono>{token}</mono>',
  buttonApply: 'Opens the apply address with the code: <mono>{token}</mono>',
  buttonNoUrl: '{name} has no apply address, so the block shows no button. Add one on the pool’s page.',
  manage: 'Manage discount codes',
  manageNote: 'Pools, their numbers and where each one is used, in Campaigns › Discount codes.',
  pass: 'Pass',
  passNote: 'Review & send checks there’s a code for every recipient.',
  warn: 'Warn',
  archivedNote: '{name} is archived: readers get no code. Pick another pool.',
  tokensGroup: 'Discount codes',
})
