import test from 'node:test'
import assert from 'node:assert/strict'
import {
  deriveMatchQueryFromFilename,
  deriveSceneTitleFromFilename,
} from './files-utils.js'

test('deriveMatchQueryFromFilename strips common release words', () => {
  const result = deriveMatchQueryFromFilename('Studio_Name_-_My.Scene_4k_60fps.mp4')

  assert.equal(result, 'Studio Name My Scene')
})

test('deriveSceneTitleFromFilename preserves apostrophes and strips technical words', () => {
  const result = deriveSceneTitleFromFilename('john_scenes_3000_4k_s02e01.mp4')

  assert.equal(result, 'John Scenes')
})
