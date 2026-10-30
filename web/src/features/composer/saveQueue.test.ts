import { describe, expect, it } from 'vitest'
import { serialQueue } from './saveQueue'

describe('serialQueue', () => {
  it('runs saves one after another, each reading what is there when its turn comes', async () => {
    const enqueue = serialQueue()
    let subject = 'Spring sale'
    const sent: string[] = []
    let release!: () => void
    const slow = new Promise<void>((r) => (release = r))
    let started!: () => void
    const running = new Promise<void>((r) => (started = r))
    // A slow save is on its way…
    const first = enqueue(async () => {
      sent.push(subject)
      started()
      await slow
      return 1
    })
    await running
    // …the person keeps typing, and the next autosave is asked for meanwhile.
    subject = 'Spring sale: 30% off'
    const second = enqueue(async () => {
      sent.push(subject)
      return 2
    })
    await Promise.resolve()
    // It waits for the first one.
    expect(sent).toEqual(['Spring sale'])
    release()
    expect(await first).toBe(1)
    expect(await second).toBe(2)
    // The second save sent the subject as it was typed, not the stale one.
    expect(sent).toEqual(['Spring sale', 'Spring sale: 30% off'])
  })

  it('keeps going after a save fails', async () => {
    const enqueue = serialQueue()
    const failed = enqueue(() => Promise.reject(new Error('offline')))
    const next = enqueue(async () => 'saved')
    await expect(failed).rejects.toThrow('offline')
    await expect(next).resolves.toBe('saved')
  })
})
