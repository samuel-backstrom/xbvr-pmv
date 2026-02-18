const state = {
  lockScrape: false,
  lastScrapeMessage: '',
  lockRescan: false,
  lastRescanMessage: '',
  lastProgressMessage: '',
  runningScrapers: [],
  serviceLogs: []
}

const mutations = {
  addServiceLog (state, payload) {
    state.serviceLogs.push(payload)
    if (state.serviceLogs.length > 500) {
      state.serviceLogs.splice(0, state.serviceLogs.length - 500)
    }
  },
  clearServiceLogs (state) {
    state.serviceLogs = []
  }
}

export default {
  namespaced: true,
  state,
  mutations
}
