<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading"></b-loading>
    <div class="content">
      <h3>{{$t("PMV Matching")}}</h3>
      <hr/>

      <p>
        Manually run PMV matching for unmatched videos.
      </p>

      <b-field>
        <b-checkbox v-model="dryRun">Dry run (no database changes)</b-checkbox>
      </b-field>

      <b-field>
        <b-checkbox v-model="refreshExisting">Refresh existing PMV scene metadata</b-checkbox>
      </b-field>

      <b-field v-if="refreshExisting" grouped group-multiline>
        <b-checkbox v-model="updateTitle">Title</b-checkbox>
        <b-checkbox v-model="updateStudio">Studio</b-checkbox>
        <b-checkbox v-model="updateSceneURL">Scene URL</b-checkbox>
        <b-checkbox v-model="updateThumbnail">Thumbnail</b-checkbox>
        <b-checkbox v-model="updateDescription">Description</b-checkbox>
      </b-field>

      <p v-if="refreshExisting" class="help">
        Refresh mode scans all scenes and updates only selected fields when a PMVHaven URL exists on the scene.
      </p>

      <b-field v-if="!refreshExisting" label="Limit">
        <b-input v-model.number="limit" type="number" min="1" max="500"></b-input>
      </b-field>

      <b-field label="Concurrency">
        <b-input v-model.number="concurrency" type="number" min="1" max="50"></b-input>
      </b-field>

      <b-field v-if="!refreshExisting" label="Path prefix">
        <b-input v-model="pathPrefix" placeholder="/mnt/g/Videos"></b-input>
      </b-field>

      <b-field v-if="!refreshExisting" label="Volume ID (optional)">
        <b-input v-model.number="volumeId" type="number" min="0" placeholder="0"></b-input>
      </b-field>

      <b-field>
        <b-button type="is-primary" @click="startTask">
          {{ refreshExisting ? 'Run PMV metadata refresh task' : 'Run PMV match task' }}
        </b-button>
      </b-field>

      <hr/>

      <p>
        Import a PMVHaven video by URL. This downloads the video into the path prefix,
        creates the PMV scene with title/thumbnail metadata, and then imports a PMVHaven
        funscript when one is available.
      </p>

      <b-field label="PMVHaven video URL">
        <b-input v-model="importURL" type="url" placeholder="https://pmvhaven.com/video/..."></b-input>
      </b-field>

      <b-field>
        <b-button type="is-primary" @click="importPMVVideo" :disabled="!importURL">
          Download PMV, create scene, and import funscript if available
        </b-button>
      </b-field>

      <hr/>

      <p>
        Import a ranked PMVHaven list or profile URL. This crawls available results,
        downloads videos in parallel, skips any video file that already exists, creates scenes,
        and imports PMVHaven funscripts when available.
      </p>

      <b-field label="PMVHaven list URL">
        <b-input v-model="importListURL" type="url" placeholder="https://pmvhaven.com/popular?period=month"></b-input>
      </b-field>

      <b-field label="Top N videos (blank = all)">
        <b-input v-model.number="importListLimit" type="number" min="1" max="100" placeholder="All"></b-input>
      </b-field>

      <b-field label="Parallel imports">
        <b-input v-model.number="importListConcurrency" type="number" min="1" max="10"></b-input>
      </b-field>

      <b-field>
        <b-button type="is-primary" @click="importPMVList" :disabled="!importListURL">
          Download PMV list
        </b-button>
      </b-field>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import { selectPMVImportVolume } from './pmvMatchingDefaults'

export default {
  name: 'PMVMatching',
  data () {
    return {
      isLoading: false,
      dryRun: true,
      limit: 20,
      concurrency: 10,
      pathPrefix: '/mnt/g/Videos',
      volumeId: 0,
      refreshExisting: false,
      updateTitle: true,
      updateStudio: true,
      updateSceneURL: true,
      updateThumbnail: true,
      updateDescription: true,
      importURL: '',
      importListURL: '',
      importListLimit: null,
      importListConcurrency: 3
    }
  },
  async mounted () {
    await this.$store.dispatch('optionsStorage/load')
    this.applyConfiguredVolumeDefaults()
  },
  methods: {
    applyConfiguredVolumeDefaults () {
      const selection = selectPMVImportVolume(
        this.$store.state.optionsStorage.items,
        this.pathPrefix,
        this.volumeId
      )
      this.pathPrefix = selection.pathPrefix
      this.volumeId = selection.volumeId
    },
    async startTask () {
      this.isLoading = true
      const searchParams = {
        dry_run: this.dryRun ? 'true' : 'false',
        limit: String(this.limit || 20),
        concurrency: String(this.concurrency || 10),
        refresh_existing: this.refreshExisting ? 'true' : 'false'
      }
      if (this.refreshExisting) {
        searchParams.update_title = this.updateTitle ? 'true' : 'false'
        searchParams.update_studio = this.updateStudio ? 'true' : 'false'
        searchParams.update_scene_url = this.updateSceneURL ? 'true' : 'false'
        searchParams.update_thumbnail = this.updateThumbnail ? 'true' : 'false'
        searchParams.update_description = this.updateDescription ? 'true' : 'false'
      } else {
        searchParams.path_prefix = this.pathPrefix || ''
        if (this.volumeId && this.volumeId > 0) {
          searchParams.volume_id = String(this.volumeId)
        }
      }

      try {
        await ky.get('/api/task/pmv-match-unmatched', { searchParams })
        this.$buefy.toast.open({
          message: this.refreshExisting ? 'PMV metadata refresh task started.' : 'PMV match task started.',
          type: 'is-success'
        })
      } catch (e) {
        this.$buefy.toast.open({
          message: this.refreshExisting ? 'Failed to start PMV metadata refresh task.' : 'Failed to start PMV match task.',
          type: 'is-danger'
        })
      } finally {
        this.isLoading = false
      }
    },
    async importPMVVideo () {
      this.isLoading = true
      try {
        const result = await ky.post('/api/task/pmv-import', {
          json: {
            url: this.importURL,
            path_prefix: this.pathPrefix || ''
          },
          timeout: false
        }).json()
        this.$buefy.toast.open({
          message: result.message || 'PMV imported successfully.',
          type: 'is-success',
          duration: 5000
        })
      } catch (e) {
        this.$buefy.toast.open({
          message: 'Failed to import PMV video.',
          type: 'is-danger'
        })
      } finally {
        this.isLoading = false
      }
    },
    async importPMVList () {
      this.isLoading = true
      try {
        const result = await ky.post('/api/task/pmv-import-list', {
          json: {
            list_url: this.importListURL,
            path_prefix: this.pathPrefix || '',
            limit: this.importListLimit || 0,
            concurrency: this.importListConcurrency || 3
          },
          timeout: false
        }).json()
        this.$buefy.toast.open({
          message: `PMV list import complete. Imported ${result.imported || 0}, skipped ${result.skipped_existing || 0}, errors ${result.errors || 0}.`,
          type: (result.errors || 0) > 0 ? 'is-warning' : 'is-success',
          duration: 7000
        })
      } catch (e) {
        this.$buefy.toast.open({
          message: 'Failed to import PMV list.',
          type: 'is-danger'
        })
      } finally {
        this.isLoading = false
      }
    }
  }
}
</script>
