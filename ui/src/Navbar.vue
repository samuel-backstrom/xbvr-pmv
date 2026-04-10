<template>
  <b-navbar :fixed-top="true" type="is-dark" class="main-navbar">
    <template slot="brand">
      <b-navbar-item>
        <h1 class="title">XBVR <small>{{currentVersion}}</small></h1>
      </b-navbar-item>
    </template>
    <template slot="start">
      <b-navbar-item tag="router-link" :to="{ path: './' }">
        {{$t('Scenes')}}
      </b-navbar-item>
      <b-navbar-item tag="router-link" :to="{ path: './actors' }">
        {{$t('Actors')}}
      </b-navbar-item>
      <b-navbar-item tag="router-link" :to="{ path: './files' }">
        {{$t('Files')}}
      </b-navbar-item>
      <b-navbar-item tag="router-link" :to="{ path: './options' }">
        {{$t('Options')}}
      </b-navbar-item>
      <b-navbar-item @click="$store.commit('overlay/showQuickFind')">
        {{$t('Quick find')}}
      </b-navbar-item>
      <b-navbar-item tag="router-link" :to="{ path: './logs' }">
        Logs
      </b-navbar-item>
    </template>
    <template slot="end">
      <b-navbar-item>
        <table style="font-size:0.9em">
          <tr v-if="Object.keys(lastRescanMessage).length !== 0">
            <th><span :class="[lockRescan ? 'pulsate' : '']">{{$t('Files')}} →</span></th>
            <td>{{lastRescanMessage.message}}</td>
          </tr>
          <tr v-if="Object.keys(lastScrapeMessage).length !== 0">
            <th><span :class="[lockScrape ? 'pulsate' : '']">{{$t('Data')}} →</span></th>
            <td>{{lastScrapeMessage.message}}</td>
          </tr>
        </table>
      </b-navbar-item>
    </template>
  </b-navbar>
</template>

<script>
import ky from 'ky'

export default {
  data () {
    return {
      currentVersion: '',
      latestVersion: ''
    }
  },
  computed: {
    lockRescan () {
      return this.$store.state.messages.lockRescan
    },
    lastRescanMessage () {
      return this.$store.state.messages.lastRescanMessage
    },
    lockScrape () {
      return this.$store.state.messages.lockScrape
    },
    lastScrapeMessage () {
      return this.$store.state.messages.lastScrapeMessage
    }
  },
  mounted () {
    ky.get('/api/options/version-check').json().then(data => {
      this.currentVersion = data.current_version
      this.latestVersion = data.latest_version

      if (data.update_notify && this.currentVersion !== 'CURRENT') {
        this.$buefy.snackbar.open({
          message: `Version ${this.latestVersion} available!`,
          type: 'is-warning',
          position: 'is-bottom-right',
          actionText: this.$t('Download now'),
          indefinite: true,
          onAction: () => {
            window.location = 'https://github.com/xbapps/xbvr/releases'
          }
        })
      }
    })
  }
}
</script>

<style scoped>
  h1 {
    display: flex;
    align-items: center;
    color: var(--text-primary) !important;
    font-weight: 700;
    letter-spacing: -0.02em;
  }

  h1 small {
    font-size: 0.45em;
    margin-left: 0.6em;
    opacity: 0.35;
    font-weight: 400;
    color: var(--text-muted) !important;
  }

  th {
    padding-right: 1em;
    color: var(--text-secondary);
    font-size: 0.85em;
  }

  td {
    color: var(--text-muted);
    font-size: 0.85em;
  }

  .pulsate {
    animation: pulsate 0.5s linear infinite;
    opacity: 0.5;
  }

  @keyframes pulsate {
    0% { opacity: 0.5; }
    50% { opacity: 1.0; }
    100% { opacity: 0.5; }
  }
</style>

<style>
  .main-navbar.navbar.is-dark {
    background-color: var(--bg-secondary) !important;
    border-bottom: 1px solid var(--border-color);
    box-shadow: 0 1px 12px rgba(0, 0, 0, 0.3);
  }
  .main-navbar .navbar-item {
    color: var(--text-secondary) !important;
    font-weight: 500;
    font-size: 0.9rem;
    transition: color var(--transition-fast), background-color var(--transition-fast);
    border-radius: var(--radius-sm);
    margin: 0 2px;
  }
  .main-navbar .navbar-item:hover,
  .main-navbar .navbar-item:focus {
    background-color: var(--bg-surface) !important;
    color: var(--text-primary) !important;
  }
  .main-navbar .navbar-item.router-link-exact-active,
  .main-navbar .navbar-item.router-link-active:not([href*="./"]) {
    color: var(--accent-primary) !important;
    background-color: rgba(108, 92, 231, 0.1) !important;
  }
  .main-navbar .navbar-burger span {
    background-color: var(--text-secondary) !important;
  }
  .main-navbar .navbar-menu.is-active {
    background-color: var(--bg-secondary) !important;
  }
</style>
