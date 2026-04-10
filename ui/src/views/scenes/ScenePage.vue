<template>
  <div class="container is-fluid scene-page">
    <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>
    <div class="columns scene-page-columns">
      <div class="column is-one-fifth scene-page-sidebar">
        <Filters/>
      </div>

      <div class="column is-four-fifths scene-page-content">
        <Details v-if="!isLoading && hasScene" :embedded="true"/>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import Details from './Details'
import Filters from './Filters'

export default {
  name: 'ScenePage',
  components: { Details, Filters },
  data () {
    return {
      isLoading: false
    }
  },
  computed: {
    hasScene () {
      return Boolean(this.$store.state.overlay.inlineDetails.scene)
    }
  },
  async created () {
    await this.loadScene()
  },
  beforeRouteUpdate (to, from, next) {
    this.loadScene(to)
      .then(() => next())
      .catch(() => next())
  },
  beforeRouteLeave (to, from, next) {
    this.$store.commit('overlay/hideInlineDetails')
    next()
  },
  methods: {
    async loadScene (route = this.$route) {
      this.isLoading = true
      try {
        if (route.query !== undefined && route.query.q !== undefined) {
          this.$store.commit('sceneList/stateFromQuery', route.query)
          await this.$store.dispatch('sceneList/load', { offset: 0 })
        }

        await Promise.all([
          this.$store.dispatch('optionsWeb/load'),
          this.$store.dispatch('optionsAdvanced/load')
        ])

        const data = await ky.get(`/api/scene/${route.params.id}`).json()
        if (data && data.id) {
          this.$store.commit('overlay/showInlineDetails', { scene: data })
          return
        }
      } catch (error) {
      } finally {
        this.isLoading = false
      }

      this.$store.commit('overlay/hideInlineDetails')
      this.$router.replace({
        name: 'scenes',
        query: route.query
      })
    }
  }
}
</script>

<style scoped>
.scene-page-sidebar,
.scene-page-content {
  display: flex;
  flex-direction: column;
}
</style>
