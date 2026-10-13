// Minimal types for opentype.js 2.0 (it ships none), covering what
// scripts/assets.ts uses to outline the README banner's text.
declare module 'opentype.js' {
  export interface PathCommand {
    type: 'M' | 'L' | 'C' | 'Q' | 'Z'
    x?: number
    y?: number
    x1?: number
    y1?: number
    x2?: number
    y2?: number
  }
  export interface Path {
    commands: PathCommand[]
    toPathData(decimalPlaces?: number): string
  }
  export interface Glyph {
    index: number
    getPath(x: number, y: number, fontSize: number): Path
  }
  export interface Font {
    unitsPerEm: number
    ascender: number
    descender: number
    charToGlyph(char: string): Glyph
  }
  export function parse(buffer: ArrayBuffer): Font
  const opentype: { parse: typeof parse }
  export default opentype
}
