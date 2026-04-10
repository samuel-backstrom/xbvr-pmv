import ky from 'ky'
import * as Handy from '@ohdoki/handy-sdk'

function parseTimeValue (value) {
  if (!value) {
    return 0
  }

  const parsed = new Date(value).getTime()
  return Number.isNaN(parsed) ? 0 : parsed
}

export async function loadHandyConfig () {
  const data = await ky.get('/api/options/state').json()
  return data?.config?.interfaces?.handy || {}
}

export async function loadHandyScriptData (fileId) {
  return ky.get(`/api/dms/file/${fileId}/raw`).text()
}

export async function sendHandyTransfer (sceneId, fileId, startTimeMs = 0, play = false) {
  return ky.post(`/api/scene/${sceneId}/handy/transfer`, {
    json: {
      file_id: fileId,
      start_time_ms: startTimeMs,
      play: play
    }
  }).json()
}

export async function stopHandyScript () {
  return ky.post('/api/options/interface/handy/remove-script', { json: {} }).json()
}

export function createHandyClient () {
  return Handy.init()
}

export function isFunscriptFile (file) {
  if (!file || !file.filename) {
    return false
  }

  return file.type === 'script' && file.filename.toLowerCase().endsWith('.funscript')
}

export function selectHandyScriptFile (scene) {
  const files = Array.isArray(scene?.file) ? [...scene.file] : []
  const scripts = files.filter(isFunscriptFile)

  scripts.sort((left, right) => {
    const selectedDelta = Number(!!right.is_selected_script) - Number(!!left.is_selected_script)
    if (selectedDelta !== 0) {
      return selectedDelta
    }

    const createdDelta = parseTimeValue(right.created_time) - parseTimeValue(left.created_time)
    if (createdDelta !== 0) {
      return createdDelta
    }

    return (right.id || 0) - (left.id || 0)
  })

  return scripts.length > 0 ? scripts[0] : null
}
