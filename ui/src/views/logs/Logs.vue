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
  background: var(--bg-card);
  border-radius: var(--radius-md);
  padding: 1em;
  border: 1px solid var(--border-color);
}

.log-line {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.85em;
  color: var(--text-primary);
  margin-bottom: 0.35em;
  line-height: 1.4;
  padding: 2px 4px;
  border-radius: 3px;
}
.log-line:hover {
  background: var(--bg-surface);
}

.log-ts {
  color: var(--text-muted);
  margin-right: 0.5em;
}

.log-level {
  display: inline-block;
  min-width: 52px;
  font-weight: 600;
  text-transform: uppercase;
  margin-right: 0.5em;
  font-size: 0.8em;
}

.log-level.is-error {
  color: var(--accent-danger);
}

.log-level.is-warning {
  color: var(--accent-warning);
}

.log-level.is-info {
  color: var(--accent-info);
}

.log-level.is-debug {
  color: var(--text-muted);
}

.log-msg {
  white-space: pre-wrap;
}
</style>
