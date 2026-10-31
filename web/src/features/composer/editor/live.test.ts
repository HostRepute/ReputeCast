import type { Editor } from 'grapesjs'
import { describe, expect, it } from 'vitest'
import { cleanLiveHtml, replaceInner, withLiveText } from './live'

describe('the words of a text being typed', () => {
  it('reads them as the model would export them', () => {
    const html = 'Hi <b class="gjs-selected strong" data-gjs-type="text" contenteditable="true">there</b>, <a href="https://acme.io" data-gjs-highlightable="true">read on</a>'
    expect(cleanLiveHtml(html)).toBe('Hi <b class="strong">there</b>, <a href="https://acme.io">read on</a>')
  })

  it('reads them in an inert document, not the page', () => {
    expect(cleanLiveHtml('Hi <img src="x" onerror="window.__ranLive = true">')).toBe('Hi <img src="x" onerror="window.__ranLive = true">')
    expect((window as unknown as { __ranLive?: boolean }).__ranLive).toBeUndefined()
  })

  it('leaves a Liquid chip as its token, not its label', () => {
    const html = 'Hi <span data-rc-liquid="{{ first_name | default: &quot;there&quot; }}" contenteditable="false">First name · fallback “there”</span>'
    expect(cleanLiveHtml(html)).toBe('Hi {{ first_name | default: "there" }}')
  })

  it('swaps the inner part of an element, whatever its attributes hold', () => {
    expect(replaceInner('<mj-text css-class="a&gt;b" padding="0 24px">stale</mj-text>', 'mj-text', 'live')).toBe('<mj-text css-class="a&gt;b" padding="0 24px">live</mj-text>')
    expect(replaceInner('<mj-text title=\'x > y\'>stale</mj-text>', 'mj-text', 'live')).toBe("<mj-text title='x > y'>live</mj-text>")
  })

  it("exports the canvas with the typed words, without touching the text's model", () => {
    class TextModel {
      get(key: string) {
        return key === 'tagName' ? 'mj-text' : undefined
      }
      toHTML() {
        return '<mj-text padding="0">Hello</mj-text>'
      }
      getView() {
        const el = document.createElement('div')
        el.innerHTML = 'Hello <b data-gjs-type="text">world</b>, typed'
        return { getChildrenContainer: () => el }
      }
    }
    const text = new TextModel()
    const editor = { getEditing: () => text } as unknown as Editor
    const exported = withLiveText(editor, () => `<mjml><mj-body>${text.toHTML()}</mj-body></mjml>`)
    expect(exported).toBe('<mjml><mj-body><mj-text padding="0">Hello <b>world</b>, typed</mj-text></mj-body></mjml>')
    // Afterwards the text exports from its model again.
    expect(text.toHTML()).toBe('<mj-text padding="0">Hello</mj-text>')
    expect(Object.prototype.hasOwnProperty.call(text, 'toHTML')).toBe(false)
    // Nothing being edited: the export as it is.
    expect(withLiveText({ getEditing: () => null } as unknown as Editor, () => 'as is')).toBe('as is')
  })
})
