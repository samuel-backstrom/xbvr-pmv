import test from 'node:test'
import assert from 'node:assert/strict'

import { selectPMVImportVolume } from './pmvMatchingDefaults.js'

test('selectPMVImportVolume picks the first configured local volume by default', () => {
  const result = selectPMVImportVolume([
    { id: 7, type: 'local', path: '/Users/samuelbackstrom/Movies/VR' },
    { id: 9, type: 'local', path: '/Volumes/Archive/PMV' }
  ])

  assert.deepEqual(result, {
    pathPrefix: '/Users/samuelbackstrom/Movies/VR',
    volumeId: 7
  })
})

test('selectPMVImportVolume keeps a valid existing selection', () => {
  const result = selectPMVImportVolume([
    { id: 7, type: 'local', path: '/Users/samuelbackstrom/Movies/VR' },
    { id: 9, type: 'local', path: '/Volumes/Archive/PMV' }
  ], '/Volumes/Archive/PMV', 9)

  assert.deepEqual(result, {
    pathPrefix: '/Volumes/Archive/PMV',
    volumeId: 9
  })
})

test('selectPMVImportVolume preserves the current value when no local volume exists', () => {
  const result = selectPMVImportVolume([], '/custom/path', 0)

  assert.deepEqual(result, {
    pathPrefix: '/custom/path',
    volumeId: 0
  })
})
