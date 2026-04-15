import { describe, expect, it } from 'bun:test'
import { slugify } from './slug.js'

describe('slugify', () => {
  const cases: Array<[string, string]> = [
    ['My Board', 'my-board'],
    ['Hello, World!', 'hello-world'],
    ['  spaces  ', 'spaces'],
    ['!!!', ''],
    ['Foo___Bar', 'foobar'],
    ['a   b', 'a-b'],
    ['--leading', 'leading'],
    ['trailing--', 'trailing'],
    ['a--b', 'a-b'],
    ['MIXEDcase', 'mixedcase'],
    ['', ''],
    ['日本語', ''],
  ]
  for (const [input, expected] of cases) {
    it(`${JSON.stringify(input)} → ${JSON.stringify(expected)}`, () => {
      expect(slugify(input)).toBe(expected)
    })
  }
})
