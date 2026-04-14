// Bundles the shell and stub renderer entrypoints for the browser.
// Run via: bun run web/shell/build.ts

import { mkdir, copyFile, rm } from 'node:fs/promises'
import { join } from 'node:path'

const root = import.meta.dir
const dist = join(root, 'dist')
const stubDist = join(dist, 'renderer-stub')

await rm(dist, { recursive: true, force: true })
await mkdir(stubDist, { recursive: true })

const results = await Bun.build({
  entrypoints: [join(root, 'src/main.ts'), join(root, 'stub/src/main.ts')],
  outdir: dist,
  target: 'browser',
  format: 'esm',
  naming: {
    entry: '[dir]/[name].[ext]',
  },
  minify: false,
  sourcemap: 'linked',
})

if (!results.success) {
  console.error('build failed:')
  for (const log of results.logs) console.error(log)
  process.exit(1)
}

// Bun puts src/main.js under src/ and stub/src/main.js under stub/src/ — move
// them to the URLs the HTML expects.
await Bun.write(Bun.file(join(dist, 'main.js')), Bun.file(join(dist, 'src/main.js')))
await Bun.write(
  Bun.file(join(stubDist, 'main.js')),
  Bun.file(join(dist, 'stub/src/main.js')),
)
await rm(join(dist, 'src'), { recursive: true, force: true })
await rm(join(dist, 'stub'), { recursive: true, force: true })

await copyFile(join(root, 'index.html'), join(dist, 'index.html'))
await copyFile(join(root, 'stub/index.html'), join(stubDist, 'index.html'))

console.log('shell build → web/shell/dist/')
