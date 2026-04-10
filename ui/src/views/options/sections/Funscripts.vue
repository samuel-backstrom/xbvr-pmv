<template>
  <div>
    <div class="content">
      <h3>{{ $t("Export funscripts") }}</h3>
      <p>
        {{$t('Here you can download a ZIP file containing a funscript for each scripted scene. The file names include scene title and scene id, as expected by DeoVR. If a scene has multiple scripts you can choose a preferred script in the scene details view. Otherwise, the most recently added script is chosen.')}}
      </p>
      <p>
        {{ $t("Note that the filenames are not compatible with DLNA.") }}
      </p>
      <p>
        {{
          $t(
            "To use this export with DeoVR: Unzip and put the files in the Interactive folder on your device."
          )
        }}
      </p>
      <p>
        {{
          $t(
            "To use this export with ScriptPlayer: Unzip and put the files in a folder of your choice. In the ScriptPlayer settings, add this folder in the Paths section, then connect to DeoVR."
          )
        }}
      </p>
      <hr />
      <p><strong>Download funscripts for DeoVR</strong></p>
      <p>
        <b-button
          type="is-primary"
          @click="exportAllFunscripts"
          :disabled="countTotal === 0"
          icon-left="download"
          >{{ $t("Download all funscripts") }} ({{ countTotal }})</b-button
        >
      </p>
      <p>
        <b-button
          type="is-primary"
          @click="exportNewFunscripts"
          :disabled="countUpdated === 0"
          icon-left="download"
          >{{ $t("Download changes since last export") }} ({{
            countUpdated
          }})</b-button
        >
      </p>
      <hr />
      <p><strong>Generate funscripts with PythonDancer</strong></p>
      <p>
        Generate `.funscript` files beside all videos that do not already have one.
        The task uses `automap`, writes a heatmap, retries with convert only if needed,
        and links the generated funscript back to the matched video scene.
      </p>
      <b-field label="Concurrency">
        <b-input v-model.number="pythonDancerConcurrency" type="number" min="1" max="4"></b-input>
      </b-field>
      <b-field label="Path prefix">
        <b-input v-model="pythonDancerPathPrefix" placeholder="/mnt/g/Videos"></b-input>
      </b-field>
      <b-field label="Volume ID (optional)">
        <b-input v-model.number="pythonDancerVolumeId" type="number" min="0" placeholder="0"></b-input>
      </b-field>
      <p>
        <b-button
          type="is-primary"
          @click="runPythonDancerTask"
          icon-left="play"
        >
          Run PythonDancer task
        </b-button>
      </p>
      <hr />
      <b-field>
        <b-switch v-model="scrapeFunscripts" type="is-default">
          <strong>Scrape for Available Funscripts</strong>
        </b-switch>
      </b-field>
      <b-field>
        <b-button type="is-primary" @click="save">Save</b-button>
      </b-field>
    </div>
  </div>
</template>

<script>
import ky from "ky";

export default {
  name: "Funscripts",
  data () {
    return {
      pythonDancerConcurrency: 1,
      pythonDancerPathPrefix: '/mnt/g/Videos',
      pythonDancerVolumeId: 0
    }
  },
  mounted() {
    this.$store.dispatch("optionsFunscripts/load");
  },
  methods: {
    exportAllFunscripts() {
      const link = document.createElement("a");
      link.href = "/api/task/funscript/export-all";
      link.click();
    },
    exportNewFunscripts() {
      const link = document.createElement("a");
      link.href = "/api/task/funscript/export-new";
      link.click();
    },
    save () {
      this.$store.dispatch('optionsFunscripts/save')
    },
    async runPythonDancerTask () {
      const searchParams = {
        concurrency: String(this.pythonDancerConcurrency || 1),
        path_prefix: this.pythonDancerPathPrefix || ''
      }
      if (this.pythonDancerVolumeId && this.pythonDancerVolumeId > 0) {
        searchParams.volume_id = String(this.pythonDancerVolumeId)
      }

      try {
        await ky.get('/api/task/funscript/python-dancer', { searchParams })
        this.$buefy.toast.open({
          message: 'PythonDancer funscript task started.',
          type: 'is-success'
        })
      } catch (e) {
        this.$buefy.toast.open({
          message: 'Failed to start PythonDancer funscript task.',
          type: 'is-danger'
        })
      }
    }
  },
  computed: {
    countTotal: function () {
      return this.$store.state.optionsFunscripts.countTotal;
    },
    countUpdated: function () {
      return this.$store.state.optionsFunscripts.countUpdated;
    },
    scrapeFunscripts: {
      get () {
        return this.$store.state.optionsFunscripts.optionsFunscripts.scrapeFunscripts
      },
      set (value) {
        this.$store.state.optionsFunscripts.optionsFunscripts.scrapeFunscripts = value
      },
    },
  },
};
</script>
