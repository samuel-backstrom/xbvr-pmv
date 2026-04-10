<template>
  <div class="modal is-active">
    <div class="modal-background"></div>
    <div class="modal-content">
      <video ref="player"
             width="640" height="640" class="video-js vjs-default-skin"
             controls playsinline autoplay>
        <source :src="sourceUrl" type="video/mp4">
      </video>
    </div>
    <button class="modal-close is-large" aria-label="close"
            @click="close()"></button>
  </div>
</template>

<script>
import ky from 'ky'
import videojs from 'video.js'
import vr from 'videojs-vr/dist/videojs-vr.min.js'
import hotkeys from 'videojs-hotkeys'
import { loadHandyConfig, selectHandyScriptFile, sendHandyTransfer, stopHandyScript } from '../../lib/handy'

export default {
  name: 'Details',
  data () {
    return {
      player: {},
      handyReady: false,
      handyScriptFileId: 0,
      handySetupPromise: null,
      handyClosed: false
    }
  },
  computed: {
    sourceUrl () {
      if (this.$store.state.overlay.player.file) {
        return '/api/dms/file/' + this.$store.state.overlay.player.file.id + '?dnt=true'
      }
      return ''
    }
  },
  mounted () {
    this.player = videojs(this.$refs.player)
    const vr = this.player.vr({
      projection: 'NONE',
      forceCardboard: false
    })

    this.player.hotkeys({
      alwaysCaptureHotkeys: true,
      volumeStep: 0.1,
      seekStep: 5,
      enableModifiersForNumbers: false,
      customKeys: {
        closeModal: {
          key: function (event) {
            return event.which === 27
          },
          handler: (player, options, event) => {
            this.close()
          }
        }
      }
    })

    this.player.on('seeked', () => {
      this.syncHandyPlayback()
    })

    this.player.on('play', () => {
      this.syncHandyPlayback()
    })

    this.player.on('pause', () => {
      this.stopHandyPlayback()
    })

    this.player.on('ended', () => {
      this.stopHandyPlayback()
    })

    this.player.on('loadedmetadata', function () {
      vr.camera.position.set(-1, 0, -1)
    })

    this.handySetupPromise = this.setupHandy()
  },
  methods: {
    async setupHandy () {
      try {
        const file = this.$store.state.overlay.player.file
        if (!file || !file.scene_id) {
          return
        }

        const handyConfig = await loadHandyConfig()
        if (!handyConfig.enabled || !handyConfig.connection_key) {
          return
        }

        if (this.handyClosed) {
          return
        }

        const scene = await ky.get(`/api/scene/${file.scene_id}`).json()
        const scriptFile = selectHandyScriptFile(scene)
        if (!scriptFile) {
          console.info('[Handy] no funscript found for file player scene', { sceneId: file.scene_id })
          return
        }

        console.info('[Handy] preparing script upload for file player', { sceneId: file.scene_id, scriptId: scriptFile.id })
        await sendHandyTransfer(file.scene_id, scriptFile.id, 0, false)
        if (this.handyClosed) {
          return
        }

        this.handyScriptFileId = scriptFile.id
        this.handyReady = true
        console.info('[Handy] script uploaded for file player', { sceneId: file.scene_id, scriptId: scriptFile.id })

        if (!this.player.paused()) {
          await this.syncHandyPlayback()
        }
      } catch (error) {
        console.warn('[Handy] file player setup failed', error)
      }
    },
    async syncHandyPlayback () {
      if (!this.handyReady || this.handyClosed || this.player.paused() || !this.handyScriptFileId) {
        return
      }

      const file = this.$store.state.overlay.player.file
      if (!file || !file.scene_id) {
        return
      }

      await sendHandyTransfer(file.scene_id, this.handyScriptFileId, Math.round(this.player.currentTime() * 1000), true)
    },
    async stopHandyPlayback () {
      if (!this.handyReady || this.handyClosed) {
        return
      }

      try {
        console.info('[Handy] stopping script for file player')
        await stopHandyScript()
        console.info('[Handy] stopped script for file player')
      } catch (error) {
        console.warn('[Handy] file player stop failed', error)
      }
    },
    async close () {
      if (this.handyReady) {
        try {
          await stopHandyScript()
        } catch (error) {
          console.warn('[Handy] file player close stop failed', error)
        }
      }
      this.handyClosed = true
      if (this.player && !this.player.paused()) {
        this.player.pause()
      }
      this.player.dispose()
      this.$store.commit('overlay/hidePlayer')
    }
  }
}
</script>

<style scoped>
  .video-js {
    margin: 0 auto;
  }
</style>
