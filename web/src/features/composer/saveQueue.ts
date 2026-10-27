/**
 * Runs async jobs one after another, in the order they were asked for. The
 * composer's saves go through one: a save asked for while another is on its
 * way waits for it, then reads what is on screen at that moment, so a
 * subject typed during a slow save is never marked saved by it (CO-2).
 */
export function serialQueue() {
  let last: Promise<unknown> = Promise.resolve()
  return <T>(run: () => Promise<T>): Promise<T> => {
    const p = last.then(run, run)
    last = p.catch(() => undefined)
    return p
  }
}
