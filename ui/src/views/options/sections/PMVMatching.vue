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

      <b-field label="Limit">
        <b-input v-model.number="limit" type="number" min="1" max="500"></b-input>
      </b-field>

      <b-field label="Concurrency">
        <b-input v-model.number="concurrency" type="number" min="1" max="50"></b-input>
      </b-field>

      <b-field label="Path prefix">
        <b-input v-model="pathPrefix" placeholder="/mnt/g/Videos"></b-input>
      </b-field>

      <b-field label="Volume ID (optional)">
        <b-input v-model.number="volumeId" type="number" min="0" placeholder="0"></b-input>
      </b-field>

      <b-field>
        <b-button type="is-primary" @click="startTask">Run PMV match task</b-button>
      </b-field>
    </div>
  </div>
</template>

<script>
import ky from 'ky'

export default {
  name: 'PMVMatching',
  data () {
    return {
      isLoading: false,
      dryRun: true,
      limit: 20,
      concurrency: 10,
      pathPrefix: '/mnt/g/Videos',
      volumeId: 0
    }
  },
  methods: {
    async startTask () {
      this.isLoading = true
      const searchParams = {
        dry_run: this.dryRun ? 'true' : 'false',
        limit: String(this.limit || 20),
        concurrency: String(this.concurrency || 10),
        path_prefix: this.pathPrefix || ''
      }
      if (this.volumeId && this.volumeId > 0) {
        searchParams.volume_id = String(this.volumeId)
      }

      try {
        await ky.get('/api/task/pmv-match-unmatched', { searchParams })
        this.$buefy.toast.open({
          message: 'PMV match task started.',
          type: 'is-success'
        })
      } catch (e) {
        this.$buefy.toast.open({
          message: 'Failed to start PMV match task.',
          type: 'is-danger'
        })
      } finally {
        this.isLoading = false
      }
    }
  }
}
</script>
