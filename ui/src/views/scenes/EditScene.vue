<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="handleEscape"
      @keyup.s="save"/>

    <div class="modal-background"></div>

    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">{{ this.scene.id == 0 ? $t('Display scene details') : $t('Edit scene details') }}</p>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>

      <section class="modal-card-body">
        <b-tabs position="is-centered" :animated="false">

          <b-tab-item :label="$t('Information')">
            <b-field :label="$t('Title')">
              <b-input type="text" v-model="scene.title" @blur="blur('title')"/>
            </b-field>

            <b-field :label="$t('Multipart scene')">
              <b-checkbox v-model="scene.is_multipart"/>
            </b-field>

            <b-field grouped group-multiline>
              <b-field :label="$t('Studio')">
                <b-input type="text" v-model="scene.studio" @blur="blur('studio')"/>
              </b-field>

              <b-field :label="$t('Site')">
                <b-input type="text" v-model="scene.site" @blur="blur('site')"/>
              </b-field>

              <b-field :label="$t('Scene URL')">
                <b-input type="text" v-model="scene.scene_url" @blur="blur('scene_url')"/>
              </b-field>
              <b-field label="PMVHaven URL">
                <b-input type="text" v-model="scene.pmvhaven_url" @blur="blur('pmvhaven_url')"/>
              </b-field>

              <b-field :label="$t('Release Date')">
                <div class="control">
                  <input type="date" class="input" v-model="scene.release_date_text"
                         @blur="blur('release_date_text')"/>
                </div>
              </b-field>

              <b-field :label="$t('Duration')">
                <b-input type="number" v-model="scene.duration" @blur="blur('duration')"/>
              </b-field>
            </b-field>

            <b-field :label="$t('Cast')">
              <b-taginput type="is-warning"
                          icon="label"
                          placeholder="Add an actor"
                          v-model="scene.castArray"
                          autocomplete
                          :allow-new="true"
                          :allow-duplicates="false"
                          :data="filteredCast"
                          @typing="getFilteredCast"
                          @blur="blur('castArray')"/>
            </b-field>

            <b-field :label="$t('Tags')">
              <b-taginput type="is-info"
                          icon="label"
                          placeholder="Add a tag"
                          v-model="scene.tagsArray"
                          autocomplete
                          :allow-new="true"
                          :allow-duplicates="false"
                          :data="filteredTags"
                          @typing="getFilteredTags"
                          @blur="blur('tagsArray')"/>
            </b-field>

            <b-field :label="$t('Description')">
              <b-input type="textarea" v-model="scene.synopsis" @blur="blur('synopsis')"/>
            </b-field>
          </b-tab-item>

          <b-tab-item :label="$t('Filenames')">
            <ListEditor :list="this.scene.files" type="files" :blurFn="() => blur('files')"/>
          </b-tab-item>

          <b-tab-item :label="$t('Gallery')">
            <GalleryEditor
              :list.sync="scene.gallery"
              :blurFn="() => blur('gallery')"
              :coverUrl="this.scene.cover_url"
              @setCover="setCoverImage"
            />
          </b-tab-item>

          <b-tab-item :label="$t('Funscripts')">
            <div class="content">
              <p>
                Force regenerate `.funscript` files for the scene videos using PythonDancer.
                Choose whether to force post-processing or skip it entirely.
              </p>

              <div v-if="videoFiles.length === 0" class="notification is-light">
                No video files are linked to this scene.
              </div>

              <div v-for="file in videoFiles" :key="file.id" class="box funscript-box">
                <div class="is-flex is-justify-content-space-between is-align-items-center funscript-row">
                  <div class="funscript-meta">
                    <strong>{{ file.filename }}</strong>
                    <p class="is-size-7 has-text-grey">{{ file.path }}</p>
                  </div>
                  <div class="buttons are-small">
                    <b-button
                      type="is-primary"
                      icon-left="pulse"
                      :loading="regenFileId === file.id && regenMode === 'always'"
                      @click="regenerateFunscript(file, 'always')"
                    >
                      Regenerate with post-processing
                    </b-button>
                    <b-button
                      type="is-light"
                      icon-left="pulse"
                      :loading="regenFileId === file.id && regenMode === 'never'"
                      @click="regenerateFunscript(file, 'never')"
                    >
                      Regenerate without post-processing
                    </b-button>
                    <b-button
                      type="is-info"
                      icon-left="folder-open"
                      :disabled="scene.id == 0"
                      @click="openFunscriptPicker(file)"
                    >
                      Select funscript
                    </b-button>
                    <b-button
                      type="is-warning"
                      icon-left="image-plus"
                      :disabled="scene.id == 0"
                      :loading="thumbFileId === file.id"
                      @click="generateThumbnail(file)"
                    >
                      Generate thumbnail
                    </b-button>
                  </div>
                </div>
              </div>
            </div>
          </b-tab-item>
        </b-tabs>

      </section>

      <footer class="modal-card-foot is-justify-content-space-between">
        <div class="buttons">
          <b-button type="is-primary" @click="save">{{ $t('Save Scene Details') }}</b-button>
          <link-stashdb-button :item="scene" objectType="scene" />
        </div>
        <b-button v-if="this.scene.id != 0" type="is-danger" outlined @click="deletescene">{{ $t('Delete Scene') }}</b-button>
      </footer>
    </div>

    <div v-if="showFunscriptPicker" class="modal is-active">
      <div class="modal-background" @click="closeFunscriptPicker"></div>
      <div class="modal-card funscript-picker-modal">
        <header class="modal-card-head">
          <p class="modal-card-title">Select funscript</p>
          <button class="delete" @click="closeFunscriptPicker" aria-label="close"></button>
        </header>
        <section class="modal-card-body">
          <b-loading :is-full-page="false" :active.sync="funscriptPickerLoading" />
          <div class="content">
            <p>
              Pick a `.funscript` for <strong>{{ funscriptPickerVideoFile ? funscriptPickerVideoFile.filename : 'the selected video' }}</strong>.
              XBVR will rename the funscript to match the video file and then match it to this scene.
            </p>
          </div>
          <b-field label="Filter filenames" label-position="on-border">
            <b-input v-model="funscriptPickerQuery" placeholder="Search within funscripts" />
          </b-field>
          <div class="mb-3">
            <b-button type="is-light" icon-left="refresh" :loading="funscriptPickerLoading" @click="loadFunscriptPickerFiles">
              Refresh list
            </b-button>
          </div>
          <b-table :data="filteredFunscriptPickerFiles" :loading="funscriptPickerLoading" paginated :per-page="8" hoverable>
            <b-table-column field="filename" label="Filename" v-slot="props">
              <strong>{{ props.row.filename }}</strong>
            </b-table-column>
            <b-table-column field="path" label="Path" v-slot="props">
              <span class="path-cell">{{ props.row.path }}</span>
            </b-table-column>
            <b-table-column field="scene_id" label="Matched" width="120" v-slot="props">
              <span>{{ props.row.scene_id || 'unmatched' }}</span>
            </b-table-column>
            <b-table-column field="_select" width="140" v-slot="props">
              <b-button
                type="is-primary"
                size="is-small"
                :loading="pickerAttachLoadingId === props.row.id"
                @click="attachFunscript(props.row)"
              >
                Select
              </b-button>
            </b-table-column>
          </b-table>
          <div v-if="!funscriptPickerLoading && filteredFunscriptPickerFiles.length === 0" class="notification is-light">
            No funscripts found for this filter.
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import GlobalEvents from 'vue-global-events'
import ListEditor from '../../components/ListEditor'
import GalleryEditor from '../../components/GalleryEditor'
import LinkStashdbButton from '../../components/LinkStashdbButton'

export default {
  name: 'EditScene',
  components: { ListEditor, GlobalEvents, GalleryEditor, LinkStashdbButton },
  data () {
    /*
    title: string,
    synopsis: string,
    release_date_text: string,
    studio: string,
    site: string,
    scene_url: string,
    cast: object[]
    tags: object[]
    images: object[]
    filenames_arr: string[]
    is_multipart: bool
     */
    const scene = Object.assign({}, this.$store.state.overlay.edit.scene)
    scene.pmvhaven_url = ''
    if (typeof scene.scene_url === 'string' && scene.scene_url.toLowerCase().includes('pmvhaven.com/video/')) {
      scene.pmvhaven_url = scene.scene_url
    }
    scene.castArray = scene.cast.map(c => c.name)
    scene.tagsArray = scene.tags.map(t => t.name)
    let images
    try {
      images = JSON.parse(scene.images)
    } catch {
      images = []
    }

    try {
      // map all scene images into the gallery list
      scene.gallery = images.map(i => i.url)
      // -If- there is -no- explicit cover_url, set the first image as cover
      if (!scene.cover_url && scene.gallery.length > 0) {
        scene.cover_url = scene.gallery[0]
      }
    } catch { 
      scene.gallery = []
    }

    try {
      // Fetch the image filenames
      scene.files = JSON.parse(scene.filenames_arr)
      if (scene.files == null) {
        scene.files = []
      }
    } catch {
      scene.files = []
    }
    return {
      scene,
      // A shallow copy won't work, need a deep copy
      source: JSON.parse(JSON.stringify(scene)),
      filteredCast: [],
      filteredTags: [],
      changesMade: false,
      regenFileId: 0,
      regenMode: '',
      thumbFileId: 0,
      showFunscriptPicker: false,
      funscriptPickerVideoFile: null,
      funscriptPickerFiles: [],
      funscriptPickerQuery: '',
      funscriptPickerLoading: false,
      pickerAttachLoadingId: 0
    }
  },
  methods: {
    getFilteredCast (text) {
      this.filteredCast = this.filters.cast.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0) &&
        !this.scene.cast.some(entry => entry.name === option.toString())
      )
    },
    getFilteredTags (text) {
      this.filteredTags = this.filters.tags.filter(option => (
        option.toString().toLowerCase().indexOf(text.toLowerCase()) >= 0) &&
        !this.scene.tags.some(entry => entry.name === option.toString())
      )
    },
    handleEscape () {
      if (this.showFunscriptPicker) {
        this.closeFunscriptPicker()
        return
      }
      this.close()
    },
    close () {
      if (this.changesMade) {
        this.$buefy.dialog.confirm({
          title: 'Close without saving',
          message: 'Are you sure you want to close before saving your changes?',
          confirmText: 'Close',
          type: 'is-warning',
          hasIcon: true,
          onConfirm: () => this.$store.commit('overlay/hideEditDetails')
        })
        return
      }
      this.$store.commit('overlay/hideEditDetails')
    },
    save () {
      // If there are images in the gallery, ensure a cover is set and gallery is normalized
      if (this.scene.gallery.length > 0) {
        // If no cover is set, use the first image as cover
        let coverUrl = this.scene.cover_url || this.scene.gallery[0];
        this.scene.cover_url = coverUrl;
        // Normalize gallery: cover first, no duplicates
        this.scene.gallery = [coverUrl, ...this.scene.gallery.filter(url => url !== coverUrl)];
      }

      // Load original image metadata (from DB) to preserve type and orientation
      let originalImages = [];
      try {
        originalImages = JSON.parse(this.source.images);
      } catch {
        originalImages = [];
      }

      // Build images metadata array
      const seen = new Set();
      const images = this.scene.gallery.reduce((arr, url) => {
        if (seen.has(url)) return arr;
        seen.add(url);
        const existing = originalImages.find(img => img.url === url);
        let type = existing?.type;
        const orientation = existing?.orientation || '';
        if (type !== 'cover' && type !== 'gallery') {
          type = (url === this.scene.cover_url) ? 'cover' : 'gallery';
        }
        arr.push({ url, type, orientation });
        return arr;
      }, []);
      this.scene.images = JSON.stringify(images);

      this.scene.filenames_arr = JSON.stringify(this.scene.files);
      this.scene.duration = String(this.scene.duration);

      // Push to backend with proper error handling
      ky.post(`/api/scene/edit/${this.scene.id}`, { json: { ...this.scene } })
        .json()
        .then(data => {
          this.$store.commit('sceneList/updateScene', data);
          this.$store.commit('overlay/showDetails', { scene: data });
          // Reset changesMade flag after successful save
          this.changesMade = false;
          this.close();
        })
        // On error, don't reset changesMade flag or close modal.User can try saving again
        .catch(error => {
          console.error('Failed to save scene:', error);
          // Show user-friendly error message
          this.$buefy.toast.open({
            message: 'Failed to save scene changes. Please try again.',
            type: 'is-danger',
            duration: 5000
          });
        });
    },
    deletescene () {
      this.$buefy.dialog.confirm({
        title: 'Delete scene',
        message: `Do you really want to delete the scene <strong>${this.scene.title}</strong> from <strong>${this.scene.studio}</strong>? If this is an existing scene, it will be re-added during the next scrape.`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {
          ky.post(`/api/scene/delete`, {json:{scene_id: this.scene.id}}).json().then(data => {
            this.$store.dispatch('sceneList/load', { offset: 0 })
            this.$store.commit('overlay/hideEditDetails')
            this.$store.commit('overlay/hideDetails')
          })
        }
      })
    },
    blur (field) {
      if (this.changesMade) return // Changes have already been made. No point to check any further
      
      // Removed 'covers'as this is now handled in the GalleryEditor component
      if (['castArray', 'tagsArray', 'files', 'gallery'].includes(field)) {
        if (this.scene[field].length !== this.source[field].length) {
          this.changesMade = true
        } else {
          for (let i = 0; i < this.scene[field].length; i++) {
            if (this.scene[field][i] !== this.source[field][i]) {
              this.changesMade = true
              break
            }
          }
        }
      } else if (this.scene[field] !== this.source[field]) {
        this.changesMade = true
      }
    },
    // Update displayed cover image in the UI
    setCoverImage (url) {
      this.scene.cover_url = url
      this.changesMade = true
    },
    openFunscriptPicker (videoFile) {
      this.funscriptPickerVideoFile = videoFile
      this.funscriptPickerQuery = ''
      this.showFunscriptPicker = true
      this.loadFunscriptPickerFiles()
    },
    closeFunscriptPicker () {
      this.showFunscriptPicker = false
      this.funscriptPickerVideoFile = null
      this.funscriptPickerFiles = []
      this.funscriptPickerQuery = ''
      this.pickerAttachLoadingId = 0
    },
    async loadFunscriptPickerFiles () {
      this.funscriptPickerLoading = true
      try {
        const data = await ky.post('/api/files/list', {
          json: {
            state: 'all',
            filename: '.funscript',
            sort: 'created_time_desc'
          }
        }).json()
        this.funscriptPickerFiles = Array.isArray(data)
          ? data.filter(file => typeof file.filename === 'string' && file.filename.toLowerCase().endsWith('.funscript'))
          : []
      } catch (error) {
        console.warn('[Handy] failed to load funscript picker files', error)
        this.funscriptPickerFiles = []
      } finally {
        this.funscriptPickerLoading = false
      }
    },
    async attachFunscript (scriptFile) {
      if (!this.funscriptPickerVideoFile || !scriptFile || this.pickerAttachLoadingId) {
        return
      }

      this.pickerAttachLoadingId = scriptFile.id
      try {
        const result = await ky.post('/api/files/attach-script', {
          json: {
            scene_id: this.scene.id,
            video_file_id: this.funscriptPickerVideoFile.id,
            script_file_id: scriptFile.id
          }
        }).json()

        this.$store.commit('sceneList/updateScene', result)
        this.$buefy.toast.open({
          message: `Matched ${scriptFile.filename} to ${this.funscriptPickerVideoFile.filename}.`,
          type: 'is-success',
          duration: 4000
        })
        this.closeFunscriptPicker()
        this.changesMade = false
        await this.refreshSceneData()
      } catch (error) {
        console.warn('[Handy] failed to attach funscript', error)
        this.$buefy.toast.open({
          message: 'Failed to attach the funscript.',
          type: 'is-danger',
          duration: 5000
        })
      } finally {
        this.pickerAttachLoadingId = 0
      }
    },
    async generateThumbnail (file) {
      if (!file || !file.id || this.thumbFileId) {
        return
      }

      this.thumbFileId = file.id
      try {
        const result = await ky.post('/api/files/generate-thumbnail', {
          json: {
            scene_id: this.scene.id,
            video_file_id: file.id
          }
        }).json()

        this.$store.commit('sceneList/updateScene', result)
        this.$buefy.toast.open({
          message: `Generated thumbnail from ${file.filename}.`,
          type: 'is-success',
          duration: 4000
        })
        await this.refreshSceneData(false)
      } catch (error) {
        console.warn('[Scene] failed to generate thumbnail', error)
        this.$buefy.toast.open({
          message: 'Failed to generate a thumbnail.',
          type: 'is-danger',
          duration: 5000
        })
      } finally {
        this.thumbFileId = 0
      }
    },
    async regenerateFunscript (file, postProcessMode) {
      if (!file || !file.id || this.regenFileId) {
        return
      }
      this.regenFileId = file.id
      this.regenMode = postProcessMode
      try {
        const result = await ky.post('/api/task/funscript/python-dancer', {
          json: {
            file_id: file.id,
            force_regenerate: true,
            post_process_mode: postProcessMode
          },
          timeout: false
        }).json()

        const item = result && result.results && result.results.length ? result.results[0] : null
        this.$buefy.toast.open({
          message: item && item.error ? item.error : 'Funscript regenerated.',
          type: item && item.error ? 'is-danger' : 'is-success',
          duration: item && item.error ? 6000 : 4000
        })

        if (!item || !item.error) {
          await this.refreshSceneData(false)
        }
      } catch (error) {
        this.$buefy.toast.open({
          message: 'Failed to regenerate funscript.',
          type: 'is-danger',
          duration: 5000
        })
      } finally {
        this.regenFileId = 0
        this.regenMode = ''
      }
    },
    async refreshSceneData (showDetails = false) {
      const data = await ky.get('/api/scene/' + this.scene.id).json()
      if (!data || !data.id) {
        return
      }

      data.pmvhaven_url = ''
      if (typeof data.scene_url === 'string' && data.scene_url.toLowerCase().includes('pmvhaven.com/video/')) {
        data.pmvhaven_url = data.scene_url
      }
      data.castArray = data.cast.map(c => c.name)
      data.tagsArray = data.tags.map(t => t.name)

      let images
      try {
        images = JSON.parse(data.images)
      } catch {
        images = []
      }
      data.gallery = images.map(i => i.url)
      if (!data.cover_url && data.gallery.length > 0) {
        data.cover_url = data.gallery[0]
      }

      try {
        data.files = JSON.parse(data.filenames_arr)
        if (data.files == null) {
          data.files = []
        }
      } catch {
        data.files = []
      }

      this.scene = data
      this.source = JSON.parse(JSON.stringify(data))
      this.$store.commit('sceneList/updateScene', data)
      if (showDetails) {
        this.$store.commit('overlay/showDetails', { scene: data })
      }
    }
  },
  computed: {
    filters () {
      return this.$store.state.sceneList.filterOpts
    },
    filteredFunscriptPickerFiles () {
      const query = this.funscriptPickerQuery.trim().toLowerCase()
      return this.funscriptPickerFiles.filter(file => {
        const filename = (file.filename || '').toLowerCase()
        const path = (file.path || '').toLowerCase()
        if (!filename.endsWith('.funscript')) {
          return false
        }
        if (!query) {
          return true
        }
        return filename.includes(query) || path.includes(query)
      })
    },
    videoFiles () {
      const files = this.scene.file || (this.$store.state.overlay.edit.scene && this.$store.state.overlay.edit.scene.file)
      if (!files) {
        return []
      }
      return files
        .filter(file => file.type === 'video')
        .slice()
        .sort((a, b) => {
          const left = Date.parse(a.created_time || '') || 0
          const right = Date.parse(b.created_time || '') || 0
          return right - left
        })
    }
  }
}
</script>

<style scoped>
.modal-card {
  width: 65%;
}

.tab-item {
  height: 40vh;
}

.funscript-box {
  margin-bottom: 1rem;
}

.funscript-row {
  gap: 1rem;
}

.funscript-meta {
  min-width: 0;
}

.funscript-picker-modal {
  width: 80%;
}

.path-cell {
  word-break: break-all;
}
</style>
