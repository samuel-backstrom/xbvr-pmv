<template>
  <div class="container">
    <div class="content">
      <div class="level">
        <div class="level-left">
          <h3 class="title is-4">Live Logs</h3>
        </div>
        <div class="level-right">
          <b-button size="is-small" @click="clearLogs">Clear</b-button>
        </div>
      </div>

      <div class="log-list">
        <div v-for="(entry, idx) in orderedLogs" :key="idx" class="log-line">
          <span class="log-ts">{{ entry.timestamp }}</span>
          <span class="log-level" :class="`is-${entry.level}`">{{ entry.level }}</span>
          <span class="log-msg">{{ entry.message }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Logs',
  computed: {
    orderedLogs () {
      return [...this.$store.state.messages.serviceLogs].reverse()
    }
  },
  methods: {
    clearLogs () {
      this.$store.commit('messages/clearServiceLogs')
    }
  }
}
</script>

<style scoped>
.log-list {
  min-height: 70vh;
  max-height: 78vh;
  overflow: auto;
  background: #111;
  border-radius: 4px;
  padding: 0.75em;
}

.log-line {
  font-family: monospace;
  font-size: 0.9em;
  color: #ddd;
  margin-bottom: 0.35em;
  line-height: 1.35;
}

.log-ts {
  color: #999;
  margin-right: 0.5em;
}

.log-level {
  display: inline-block;
  min-width: 52px;
  font-weight: 600;
  text-transform: uppercase;
  margin-right: 0.5em;
}

.log-level.is-error {
  color: #ff5a5a;
}

.log-level.is-warning {
  color: #ffbf47;
}

.log-level.is-info {
  color: #6ec1ff;
}

.log-level.is-debug {
  color: #9f9f9f;
}

.log-msg {
  white-space: pre-wrap;
}
</style>
