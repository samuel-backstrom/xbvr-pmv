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
  video_width: number
  video_height: number
  video_bitrate: number
  video_avg_frame_rate_val: number
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
  const qs = new URLSearchParams(
    Object.fromEntries(Object.entries(params).map(([k, v]) => [k, String(v)]))
  ).toString()
  return request<{ scenes: Scene[]; results: number }>(`/api/scene/list?${qs}`)
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
  const qs = new URLSearchParams(
    Object.fromEntries(Object.entries(params).map(([k, v]) => [k, String(v)]))
  ).toString()
  return request<{ results: Actor[]; total: number }>(`/api/actor/list?${qs}`)
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
export function getFiles() {
  return request<SceneFile[]>('/api/files/list')
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
