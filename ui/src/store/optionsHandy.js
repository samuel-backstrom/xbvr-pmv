import ky from 'ky'

const state = {
  loading: false,
  handy: {
    enabled: false,
    connection_key: '',
    latency_ms: 0
  },
  scriptActionLoading: false,
  status: {
    loading: false,
    connected: false,
    connection_key_set: false,
    mode: '',
    message: '',
    error: '',
    checked_at: null
  }
}

const mutations = {}

const actions = {
  async load ({ state }) {
    state.loading = true
    return ky.get('/api/options/state')
      .json()
      .then(data => {
        state.handy.enabled = data.config.interfaces.handy.enabled
        state.handy.connection_key = data.config.interfaces.handy.connection_key
        state.handy.latency_ms = data.config.interfaces.handy.latency_ms
        state.loading = false
      })
      .catch(() => {
        state.loading = false
      })
  },
  async save ({ state }) {
    state.loading = true
    return ky.put('/api/options/interface/handy', { json: { ...state.handy } })
      .json()
      .then(data => {
        state.handy.enabled = data.enabled
        state.handy.connection_key = data.connection_key
        state.handy.latency_ms = data.latency_ms
        state.loading = false
      })
      .catch(() => {
        state.loading = false
      })
  },
  async refreshStatus ({ state }) {
    state.status.loading = true
    return ky.get('/api/options/interface/handy/status')
      .json()
      .then(data => {
        state.status.connected = data.connected
        state.status.connection_key_set = data.connection_key_set
        state.status.mode = data.mode || ''
        state.status.message = data.message || ''
        state.status.error = data.error || ''
        state.status.checked_at = data.checked_at || null
        state.status.loading = false
      })
      .catch(err => {
        state.status.connected = false
        state.status.error = err?.message || 'Failed to load status'
        state.status.message = ''
        state.status.checked_at = null
        state.status.loading = false
      })
  },
  async removeScript ({ state, dispatch }) {
    state.scriptActionLoading = true
    return ky.post('/api/options/interface/handy/remove-script', { json: {} })
      .json()
      .then(() => dispatch('refreshStatus'))
      .catch(() => {})
      .finally(() => {
        state.scriptActionLoading = false
      })
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
