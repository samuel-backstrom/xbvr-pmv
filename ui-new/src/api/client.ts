const BASE = ''

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json()
}

export interface Scene {
  id: number
  scene_id: string
  title: string
  site: string
  cover_url: string
  scene_url: string
  members_url: string
  release_date: string
  duration: number
  synopsis: string
  star_rating: number
  is_available: boolean
  is_watched: boolean
  is_scripted: boolean
  is_hidden: boolean
  has_preview: boolean
  cast: Actor[]
  tags: Tag[]
  file: SceneFile[]
  cuepoints: Cuepoint[]
  score?: number
}

export interface Actor {
  id: number
  name: string
  image_url: string
  image_arr: string
  birth_date: string
  nationality: string
  ethnicity: string
  height: number
  weight: number
  cup_size: string
  band_size: number
  waist_size: number
  hip_size: number
  breast_type: string
  eye_color: string
  hair_color: string
  star_rating: number
  scene_rating_average: number
  avail_count: number
  count: number
  scenes: { id: number; is_available: number }[]
}

export interface ActorSearchResult {
  results: number
  actors: Actor[]
  count_any?: number
  count_available?: number
  count_downloaded?: number
  count_not_downloaded?: number
  count_hidden?: number
  offset?: number
}

export interface Tag {
  id: number
  name: string
  count: number
}

export interface SceneFile {
  id: number
  path: string
  filename: string
  size: number
  type: string
  scene_id: number
  video_width: number
  video_height: number
  video_bitrate: number
  video_avg_frame_rate_val: number
  duration: number
  has_heatmap: boolean
  is_selected_script: boolean
  created_time: string
}

export interface Cuepoint {
  id: number
  name: string
  time_start: number
}

export interface FilterOptions {
  cast: string[]
  sites: string[]
  tags: string[]
  attributes: string[]
  release_month: string[]
  volumes: { id: number; path: string }[]
}

export interface SceneSearchResult {
  results: number
  scenes: Scene[]
}

// Scenes
export function getScenes(params: Record<string, any> = {}) {
  const sortField = String(params.sort || '')
  const sortDir = String(params.order || 'desc')

  const sortMap: Record<string, string> = {
    release_date_text: sortDir === 'asc' ? 'release_asc' : 'release_desc',
    added_date: sortDir === 'asc' ? 'added_asc' : 'added_desc',
    star_rating: sortDir === 'asc' ? 'rating_asc' : 'rating_desc',
    scene_rating: sortDir === 'asc' ? 'rating_asc' : 'rating_desc',
    duration: sortDir === 'asc' ? 'release_asc' : 'release_desc',
    title: sortDir === 'asc' ? 'title_asc' : 'title_desc',
    random: 'random',
  }

  const body: Record<string, any> = {
    limit: params.limit,
    offset: params.offset,
    sort: sortMap[sortField] || 'release_desc',
  }

  if (Array.isArray(params.sites) && params.sites.length > 0) body.sites = params.sites
  if (Array.isArray(params.tags) && params.tags.length > 0) body.tags = params.tags
  if (Array.isArray(params.cast) && params.cast.length > 0) body.cast = params.cast
  if (params.list) body.lists = [params.list]

  if (params.is_available === true) {
    body.isAvailable = true
  }
  if (params.is_accessible === true) {
    body.isAccessible = true
  }

  if (params.is_watched === true || params.is_watched === false) {
    body.isWatched = params.is_watched
  }

  if (params.is_scripted) {
    body.attributes = [...(body.attributes || []), 'Is Scripted']
  }

  if (Number(params.rating) > 0) {
    const minRating = Math.min(5, Math.max(0, Number(params.rating)))
    const ratingAttrs = []
    for (let rating = minRating; rating <= 5; rating += 0.5) {
      ratingAttrs.push(`Rating ${rating}`)
    }
    body.attributes = [...(body.attributes || []), ...ratingAttrs]
  }

  return request<{ scenes: Scene[]; results: number }>('/api/scene/list', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getScene(id: number) {
  return request<Scene>(`/api/scene/${id}`)
}

export function getSceneFilters() {
  return request<FilterOptions>('/api/scene/filters')
}

export function searchScenes(q: string) {
  return request<SceneSearchResult>(`/api/scene/search?q=${encodeURIComponent(q)}`)
}

export function rateScene(id: number, rating: number) {
  return request<void>(`/api/scene/rate/${id}`, {
    method: 'POST',
    body: JSON.stringify({ rating }),
  })
}

export function toggleSceneList(sceneId: string, list: string) {
  return request<Scene>(`/api/scene/toggle?scene_id=${sceneId}&list=${list}`)
}

// Actors
export function getActors(params: Record<string, any> = {}) {
  const sortField = String(params.sort || '')
  const sortDir = String(params.order || 'desc')

  const sortMap: Record<string, string> = {
    name: sortDir === 'asc' ? 'name_asc' : 'name_desc',
    scene_count: sortDir === 'asc' ? 'name_asc' : 'scene_count_desc',
    star_rating: sortDir === 'asc' ? 'rating_asc' : 'rating_desc',
    added_date: sortDir === 'asc' ? 'added_asc' : 'added_desc',
    birth_date: sortDir === 'asc' ? 'birthday_asc' : 'birthday_desc',
    random: 'random',
  }

  const body: Record<string, any> = {
    limit: params.limit,
    offset: params.offset,
    sort: sortMap[sortField] || 'name_asc',
  }

  if (Array.isArray(params.lists) && params.lists.length > 0) body.lists = params.lists
  if (Array.isArray(params.cast) && params.cast.length > 0) body.cast = params.cast
  if (Array.isArray(params.sites) && params.sites.length > 0) body.sites = params.sites
  if (Array.isArray(params.tags) && params.tags.length > 0) body.tags = params.tags
  if (Array.isArray(params.attributes) && params.attributes.length > 0) body.attributes = params.attributes
  if (params.jumpTo) body.jumpTo = params.jumpTo
  if (params.dlState) body.dlState = params.dlState

  const minRating = Number(params.min_rating ?? params.rating ?? 0)
  if (minRating > 0) body.min_rating = minRating

  const maxRating = Number(params.max_rating)
  if (!Number.isNaN(maxRating) && maxRating > 0) body.max_rating = maxRating

  const minSceneRating = Number(params.min_scene_rating)
  if (!Number.isNaN(minSceneRating) && minSceneRating > 0) body.min_scene_rating = minSceneRating

  const maxSceneRating = Number(params.max_scene_rating)
  if (!Number.isNaN(maxSceneRating) && maxSceneRating > 0) body.max_scene_rating = maxSceneRating

  return request<ActorSearchResult>('/api/actor/list', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getActor(id: number) {
  return request<Actor>(`/api/actor/${id}`)
}

export function getActorFilters() {
  return request<FilterOptions>('/api/actor/filters')
}

export function rateActor(id: number, rating: number) {
  return request<void>(`/api/actor/rate/${id}`, {
    method: 'POST',
    body: JSON.stringify({ rating }),
  })
}

// Files
export function getFiles(params: Record<string, any> = {}) {
  const body: Record<string, any> = {}
  if (params.state) body.state = params.state
  if (params.sort) body.sort = params.sort
  if (Array.isArray(params.resolutions) && params.resolutions.length > 0) body.resolutions = params.resolutions
  if (Array.isArray(params.framerates) && params.framerates.length > 0) body.framerates = params.framerates
  if (Array.isArray(params.bitrates) && params.bitrates.length > 0) body.bitrates = params.bitrates
  if (params.filename) body.filename = params.filename
  if (Array.isArray(params.createdDate) && params.createdDate.length > 0) body.createdDate = params.createdDate

  return request<SceneFile[]>('/api/files/list', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function matchFile(fileId: number, sceneId: string) {
  return request<void>('/api/files/match', {
    method: 'POST',
    body: JSON.stringify({ file_id: fileId, scene_id: sceneId }),
  })
}

export function unmatchFile(fileId: number) {
  return request<void>('/api/files/unmatch', {
    method: 'POST',
    body: JSON.stringify({ file_id: fileId }),
  })
}

export function deleteFile(fileId: number) {
  return request<void>(`/api/files/file/${fileId}`, {
    method: 'DELETE',
  })
}

export function createCustomScene(req: {
  title: string
  id?: string
  filename?: string
  pmvhaven_url?: string
}) {
  return request<Scene>('/api/scene/create', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// Storage / Volumes
export interface Volume {
  id: number
  type: string
  path: string
  last_scan: string
  is_available: boolean
  is_enabled: boolean
  file_count: number
  unmatched_count: number
  total_size: number
}

export interface StorageResponse {
  volumes: Volume[]
  match_ohash: boolean
  video_ext: string[]
  forbidden_video_ext: string[]
  default_video_ext: string[]
}

export function getStorage() {
  return request<StorageResponse>('/api/options/storage')
}

export function addStorage(data: { path?: string; token?: string; type: string }) {
  return request<void>('/api/options/storage', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function removeStorage(id: number) {
  return request<void>(`/api/options/storage/${id}`, { method: 'DELETE' })
}

export function saveStorageOptions(options: { match_ohash: boolean; video_ext: string[] }) {
  return request<void>('/api/options/storage', {
    method: 'PUT',
    body: JSON.stringify(options),
  })
}

export function rescanAll() {
  return fetch('/api/task/rescan')
}

export function rescanVolume(id: number) {
  return fetch(`/api/task/rescan/${id}`)
}

// Options
export function getVersion() {
  return request<{ current_version: string; latest_version: string }>('/api/options/version-check')
}

// Image proxy
export function imageUrl(url: string, size = '700x') {
  if (!url) return '/ui/images/blank.png'
  if (!url.startsWith('http')) return url
  return `/img/${size}/${encodeURI(url)}`
}

// PMV
export interface PMVMatchBatchRequest {
  dry_run: boolean
  limit?: number
  concurrency?: number
  volume_id?: number
  path_prefix?: string
  refresh_existing?: boolean
  update_title?: boolean
  update_studio?: boolean
  update_scene_url?: boolean
  update_thumbnail?: boolean
  update_description?: boolean
}

export interface PMVImportRequest {
  url?: string
  list_url?: string
  path_prefix?: string
  limit?: number
  concurrency?: number
}

export interface PMVImportResult {
  url: string
  scene_url?: string
  media_url?: string
  downloaded_path?: string
  file_id?: number
  scene_id?: string
  funscript_generated?: boolean
  funscript_downloaded?: boolean
  skipped?: boolean
  message?: string
}

export interface PMVImportBatchItem {
  rank: number
  url: string
  result?: PMVImportResult
  error?: string
}

export interface PMVImportBatchResult {
  list_url: string
  requested: number
  queued: number
  imported: number
  skipped_existing: number
  funscripts_generated: number
  errors: number
  results: PMVImportBatchItem[]
}

export function startPMVMatchTask(opts: {
  dryRun: boolean
  refreshExisting: boolean
  limit: number
  concurrency: number
  pathPrefix: string
  volumeId: number
  updateTitle: boolean
  updateStudio: boolean
  updateSceneURL: boolean
  updateThumbnail: boolean
  updateDescription: boolean
}) {
  const params = new URLSearchParams()
  params.set('dry_run', String(opts.dryRun))
  params.set('limit', String(opts.limit || 20))
  params.set('concurrency', String(opts.concurrency || 10))
  params.set('refresh_existing', String(opts.refreshExisting))
  if (opts.refreshExisting) {
    params.set('update_title', String(opts.updateTitle))
    params.set('update_studio', String(opts.updateStudio))
    params.set('update_scene_url', String(opts.updateSceneURL))
    params.set('update_thumbnail', String(opts.updateThumbnail))
    params.set('update_description', String(opts.updateDescription))
  } else {
    if (opts.pathPrefix) params.set('path_prefix', opts.pathPrefix)
    if (opts.volumeId > 0) params.set('volume_id', String(opts.volumeId))
  }
  return fetch(`/api/task/pmv-match-unmatched?${params.toString()}`)
}

export function importPMVVideo(req: PMVImportRequest) {
  return request<PMVImportResult>('/api/task/pmv-import', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

export function importPMVList(req: PMVImportRequest) {
  return request<PMVImportBatchResult>('/api/task/pmv-import-list', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}
