<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading" />
    <div class="content">
      <h3>The Handy</h3>
      <hr />
      <p>
        Connect XBVR's player to The Handy over Wi-Fi using the Handy SDK. XBVR uploads the current scene's
        selected `.funscript` when playback starts, then keeps the device aligned while you play and seek.
      </p>
      <p>
        The Handy connection key is required. Leave this disabled if you do not want XBVR to touch the device.
      </p>
      <hr />
      <b-field label="Enabled">
        <b-switch v-model="enabled">
          Enabled
        </b-switch>
      </b-field>
      <b-field label="Connection Key" label-position="on-border">
        <b-input v-model="connectionKey" :disabled="!enabled" type="password" placeholder="Connection key from Handy settings" />
      </b-field>
      <b-field label="Latency Offset (ms)" label-position="on-border">
        <b-numberinput v-model="latencyMs" :disabled="!enabled" :min="-5000" :max="5000" controls-position="compact" />
      </b-field>
      <p class="help">
        Positive values advance the Handy, negative values delay it.
      </p>
      <hr />
      <h4>Status</h4>
      <b-field label="Connection">
        <b-tag :type="statusTagType" rounded>
          {{ statusLabel }}
        </b-tag>
      </b-field>
      <b-field label="Mode">
        <span>{{ status.mode || 'unknown' }}</span>
      </b-field>
      <b-field label="Message">
        <span>{{ status.message || status.error || 'No status loaded yet' }}</span>
      </b-field>
      <b-field label="Last checked">
        <span>{{ lastCheckedLabel }}</span>
      </b-field>
      <b-field>
        <b-button type="is-primary" :loading="status.loading" icon-left="refresh" @click="refreshStatus">
          Refresh status
        </b-button>
      </b-field>
      <b-field>
        <b-button type="is-warning" :loading="scriptActionLoading" :disabled="!status.connected || scriptActionLoading" icon-left="stop" @click="removeScript">
          Remove script
        </b-button>
      </b-field>
      <b-field>
        <b-button type="is-primary" @click="save">Save</b-button>
      </b-field>
    </div>
  </div>
</template>

<script>
export default {
  name: 'InterfaceHandy',
  mounted () {
    this.$store.dispatch('optionsHandy/load')
    this.refreshStatus()
  },
  methods: {
    save () {
      this.$store.dispatch('optionsHandy/save').then(() => {
        this.refreshStatus()
      })
    },
    refreshStatus () {
      this.$store.dispatch('optionsHandy/refreshStatus')
    },
    removeScript () {
      this.$store.dispatch('optionsHandy/removeScript')
    }
  },
  computed: {
    isLoading () {
      return this.$store.state.optionsHandy.loading
    },
    enabled: {
      get () {
        return this.$store.state.optionsHandy.handy.enabled
      },
      set (value) {
        this.$store.state.optionsHandy.handy.enabled = value
      }
    },
    connectionKey: {
      get () {
        return this.$store.state.optionsHandy.handy.connection_key
      },
      set (value) {
        this.$store.state.optionsHandy.handy.connection_key = value
      }
    },
    latencyMs: {
      get () {
        return this.$store.state.optionsHandy.handy.latency_ms
      },
      set (value) {
        this.$store.state.optionsHandy.handy.latency_ms = value
      }
    },
    status () {
      return this.$store.state.optionsHandy.status
    },
    scriptActionLoading () {
      return this.$store.state.optionsHandy.scriptActionLoading
    },
    statusLabel () {
      if (!this.status.connection_key_set) {
        return 'No connection key'
      }
      return this.status.connected ? 'Connected' : 'Disconnected'
    },
    statusTagType () {
      if (!this.status.connection_key_set) {
        return 'is-warning'
      }
      return this.status.connected ? 'is-success' : 'is-danger'
    },
    lastCheckedLabel () {
      if (!this.status.checked_at) {
        return 'Never'
      }
      return new Date(this.status.checked_at).toLocaleString()
    }
  }
}
</script>
