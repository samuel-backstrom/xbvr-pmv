<template>
  <div :class="containerClass">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
      @keydown.left="handleLeftArrow"
      @keydown.right="handleRightArrow"
      @keydown.o="prevScene"
      @keydown.p="nextScene"
      @keydown.f="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'favourite'})"
      @keydown.exact.w="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watchlist'})"
      @keydown.shift.w="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watched'})"
      @keydown.t="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'trailerlist'})"
      @keydown.e="$store.commit('overlay/editDetails', {scene: item})"
      @keydown.s="$store.commit('overlay/showSearchStashdbScenes', {scene: item})"
      @keydown.g="toggleGallery"
      @keydown.48="setRating(0)"
    />

    <div v-if="!embedded" class="modal-background"></div>

    <div class="modal-card">
      <section class="modal-card-body">
        <div :class="layoutClass">

          <div :class="mediaColumnClass">
            <b-tabs v-model="activeMedia" position="is-centered" :animated="false">

              <b-tab-item label="Gallery">
                <b-carousel v-model="carouselSlide" @change="scrollToActiveIndicator" :autoplay="false" :indicator-inside="false">
                  <b-carousel-item v-for="(carousel, i) in images" :key="i">
                    <div class="image is-1by1 is-full"
                         v-bind:style='{backgroundImage: `url("${getImageURL(carousel.url, "700,fit")}")`, backgroundSize: "contain", backgroundPosition: "center", backgroundRepeat: "no-repeat"}'></div>
                  </b-carousel-item>
                  <template slot="indicators" slot-scope="props">
                      <span class="al image" style="width:max-content;">
                        <vue-load-image>
                          <img slot="image" :src="getIndicatorURL(props.i)" style="height:40px;"/>
                          <img slot="preloader" src="/ui/images/blank.png" style="height:40px;"/>
                          <img slot="error" src="/ui/images/blank.png" style="height:40px;"/>
                        </vue-load-image>
                      </span>
                  </template>
                </b-carousel>
              </b-tab-item>

              <b-tab-item label="Player" v-if="!displayingAlternateSource">
                <video ref="player" class="video-js vjs-default-skin" controls playsinline preload="none"/>
                <b-field position="is-centered">
                  <b-field>
                    <b-tooltip v-for="(skipBack, i) in skipBackIntervals" class="is-size-7" :key="i" :active="skipBack == lastSkipBackInterval ? true : false" :label="$t('Keyboard shortcut: Left Arrow')"
                        position="is-top" type="is-primary is-light" >
                    <b-button class="tag is-small is-outlined is-info is-light"  @click="playerStepBack(skipBack)">
                      <b-icon v-if="skipBack == lastSkipBackInterval" pack="mdi" icon="arrow-left-thin" size="is-small"></b-icon> {{ skipBack }}</b-button>
                    </b-tooltip>
                  </b-field>
                  <b-field style="margin-left:1em">
                    <b-tooltip v-for="(skipForward, i) in skipForwardIntervals" :key="i" :active="skipForward == lastSkipFowardInterval ? true : false" :label="$t('Keyboard shortcut: Right Arrow')"
                        position="is-top" type="is-primary is-light" >
                    <b-button class="tag is-small is-outlined is-info is-light" @click="playerStepForward(skipForward)">
                      <b-icon v-if="skipForward == lastSkipFowardInterval" pack="mdi" icon="arrow-right-thin" size="is-small"></b-icon> +{{ skipForward }}</b-button>
                    </b-tooltip>
                  </b-field>
                </b-field>
             </b-tab-item>

            </b-tabs>

          </div>

          <div :class="infoColumnClass">

            <div class="block-info block">
              <div class="content">
                <h3>
                  <span v-if="item.title">{{ item.title }}</span>
                  <span v-else class="missing">(no title)</span>                  
                  <small class="is-pulled-right">
                    {{ format(parseISO(item.release_date), "yyyy-MM-dd") }}
                  </small>
                </h3>
                <div class="columns">
                  <div class="column pb-0">
                    <small>
                      <a :href="item.scene_url" target="_blank" rel="noreferrer">{{ item.site }}</a>
                      <br v-if="item.members_url != ''"/>
                      <a v-if="item.members_url != ''" :href="item.members_url" target="_blank" rel="noreferrer"><b-icon pack="mdi" icon="link-lock" custom-size="mdi-18px"/>Members Link</a>
                    </small>
                  </div>
                  <div class="column pb-0">
                    <small v-if="item.duration" class="is-pulled-right">{{ item.duration }} minutes</small>
                  </div>
                </div>
                <div class="columns is-vcentered">
                  <div class="column pt-0">
                    <b-field v-if="!displayingAlternateSource">
                      <star-rating :key="item.id" v-model="item.star_rating" :rating="item.star_rating" @rating-selected="setRating"
                                   :increment="0.5" :star-size="20" :show-rating="false" />
                      <b-tooltip :label="$t('Reset Rating')" position="is-right" :delay="250">
                        <b-icon pack="mdi" icon="autorenew" size="is-small" @click.native="setRating(0)" style="padding-left: 1em;padding-top: .5em;"/>
                      </b-tooltip>
                    </b-field>
                    <b-field v-if="displayingAlternateSource">
                      <strong>Linked scene, Not an XBVR Scene</strong>
                    </b-field>
                  </div>
                  <div class="column pt-0">
                    <div class="is-flex is-pulled-right" style="gap: 0.25rem">
                      <a class="button is-primary is-outlined is-small" @click="searchAlternateSourceScene()" title="Search for a different scene" v-if="displayingAlternateSource">
                        <b-icon pack="mdi" icon="movie-search-outline" size="is-small"/>
                      </a>
                      <a class="button is-primary is-outlined is-small" @click="scrapeScene()" title="Scrape and create an XBVR scene (not a link)" v-if="displayingAlternateSource">
                        <b-icon pack="mdi" icon="plus" size="is-medium"/>
                      </a>
                      <a class="button is-primary is-outlined is-small" @click="refreshExtRef()" title="Removes the scene.  Rescrape to refresh the scene data and relink" v-if="displayingAlternateSource">
                        <b-icon pack="mdi" icon="refresh" size="is-small"/>
                      </a>
                      <a class="button is-danger is-outlined is-small" @click="flagExtRefDeleted()" title="Unlinks the scene. It cannot be relinked to any scene. This cannot be undone" v-if="displayingAlternateSource">
                        <b-icon pack="mdi" icon="delete" size="is-small"/>
                      </a>
                      <hidden-button :item="item" v-if="!displayingAlternateSource"/>
                      <watchlist-button :item="item" v-if="!displayingAlternateSource"/>
                      <trailerlist-button :item="item" v-if="!displayingAlternateSource"/>
                      <favourite-button :item="item" v-if="!displayingAlternateSource"/>
                      <wishlist-button :item="item" v-if="!displayingAlternateSource"/>
                      <watched-button :item="item" v-if="!displayingAlternateSource"/>
                      <edit-button :item="item"/>
                      <refresh-button :item="item" v-if="!displayingAlternateSource"/>
                      <rescrape-button :item="item" v-if="!displayingAlternateSource"/>
                      <link-stashdb-button :item="item" objectType="scene" />
                    </div>
                  </div>
                </div>
                <div class="image-row is-flex is-pulled-right" v-if="getAlternateSceneSources != 0">
                  <div v-for="(altsrc, idx) in alternateSourcesWithTitles" :key="idx" class="altsrc-image-wrapper" @click="showExtRefScene(altsrc)">
                    <b-tooltip type="is-light" :label="altsrc.title" :delay="100" append-to-body>
                      <vue-load-image>
                        <img slot="image" :src="getImageURL(altsrc.site_icon)" alt="Image" width="28px" />                        
                        <b-icon slot="error" pack="mdi" icon="link" size="is-small" />
                      </vue-load-image>
                    </b-tooltip>
                  </div>
                </div>
              </div>
            </div>

            <div class="image-row" v-if="activeTab != 1 && !displayingAlternateSource">
              <div v-for="(image, idx) in castimages" :key="idx" class="image-wrapper">
                <b-tooltip  type="is-light" :label="image.actor_label"  :delay=100>
                  <vue-load-image>
                    <img slot="image" :src="getImageURL(image.src)" alt="Image" class="thumbnail" @mouseover="showTooltip(idx)" @mouseout="hideTooltip(idx)" @click='showActorDetail([image.actor_id])' />
                    <img slot="preloader" :src="getImageURL('https://i.stack.imgur.com/kOnzy.gif')" style="height: 50px;display: block;margin-left:auto;margin-right: auto;" @click='showCastScenes([image.actor_name])' />
                    <img slot="error" src="/ui/images/blank_female_profile.png" width="80" @click='showActorDetail([image.actor_id])' />
                  </vue-load-image>
                </b-tooltip>

                <div v-if="image.visible" class="tooltip">
                  <img :src="getImageURL(image.src)" alt="Tooltip Image" />
                </div>
              </div>
            </div>

            <div class="block-tags block" v-if="activeTab != 1">
              <b-taglist>
                <span v-for="(c, idx) in item.cast" :key="'cast' + idx" >
                  <a class="tag is-warning is-small" @click='showCastScenes([c.name])' :style="showOpenInNewWindow ? 'margin-right: 0;': 'margin-right: .5em;'" >{{ c.name }} ({{ c.avail_count }}/{{ c.count }})</a>
                  <a v-if="showOpenInNewWindow" class="tag is-warning is-small" :href='getCastScenesUrl([c.name])' target="_blank" style="margin-right: 0.5em;"><b-icon pack="mdi" icon="open-in-new" size="is-small"></b-icon></a>
                </span>
                <span>
                  <a @click='showSiteScenes([item.site])' class="tag is-primary is-small" :style="showOpenInNewWindow ? 'margin-right: 0;': 'margin-right: .5em;'">{{ item.site }}</a>
                  <a v-if="showOpenInNewWindow" class="tag is-primary is-small" :href='getSiteScenesUrl([item.site])' target="_blank" style="margin-right: 0.5em;"><b-icon pack="mdi" icon="open-in-new" size="is-small"></b-icon></a>
                </span>
                <span v-for="(tag, idx) in item.tags" :key="'tag' + idx">
                  <a  @click='showTagScenes([tag.name])' class="tag is-info is-small" :style="showOpenInNewWindow ? 'margin-right: 0;': 'margin-right: .5em;'">{{ tag.name }} ({{ tag.count }})</a>
                  <a v-if="showOpenInNewWindow" class="tag is-info is-small" :href='getTagScenesUrl([tag.name])' target="_blank" style="margin-right: 0.5em;"><b-icon pack="mdi" icon="open-in-new" size="is-small"></b-icon></a>
                </span>
              </b-taglist>              
            </div>

            <div class="block-tags block" v-if="activeTab == 1">
             <b-taglist>
              <b-tooltip  type="is-danger" :label="disableSaveMsg()" position="is-right" :delay=250 :active="disableSaveButtons()">
                <b-button @click="updateCuepoint(false)" class="tag is-info is-small is-warning" accesskey="a" :disabled="disableSaveButtons()" >
                  <u>A</u>dd New
                </b-button>
              </b-tooltip>
                <b-button @click="vidPosition = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" class="tag is-info is-small is-warning" accesskey="t">Current <u>T</u>ime</b-button>
              <b-tooltip type="is-danger" :label="$t(disableSaveMsg())" position="is-right" :delay=250 :active="disableSaveButtons()">
                <b-button v-if="currentCuepointId > 0" @click="updateCuepoint(true)" class="tag is-info is-small is-warning" accesskey="s"
                  :disabled="disableSaveButtons()" >
                  <u>S</u>ave Edit
                </b-button>
              </b-tooltip>
                <b-button v-if="cuepointName!=''" @click='cuepointName=""' class="tag is-info is-small is-warning" >Clear Cuepoint Name</b-button>
                <b-button v-if="tagAct!=''" @click='setCuepointName("")' class="tag is-info is-small is-warning" accesskey="c"><u>C</u>lear Action</b-button>
              </b-taglist>
            </div>

            <div class="is-divider" data-content="Cuepoint Positions" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(c, idx) in cuepointPositionTags.slice(1)" :key="'pos' + idx" @click='setCuepointName([c])' class="tag is-info is-small">{{c}}</b-button>
              </b-taglist>
            </div>
            <div class="is-divider" data-content="Default Cuepoint Actions" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(c, idx) in cuepointActTags.slice(1)" :key="'action' + idx" @click='setCuepointName([c])' class="tag is-info is-small">{{c}}</b-button>
              </b-taglist>
            </div>
            <div class="is-divider" data-content="Cast Cuepoints" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(c, idx) in item.cast" :key="'cast' + idx" @click='setCuepointName([c.name])' class="tag is-info is-small">{{c.name}}</b-button>
              </b-taglist>
            </div>
            <div class="is-divider" data-content="Scene Tag Cuepoints" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(tag, idx) in item.tags" :key="'tag' + idx" @click='setCuepointName([tag.name])'
                   class="tag is-info is-small">{{ tag.name }}</b-button>
              </b-taglist>
            </div>


            <div class="block-opts block">
              <b-tabs v-model="activeTab" :animated="false">

                <b-tab-item :label="`Files (${fileCount})`" v-if="!displayingAlternateSource">
                  <div class="block-tab-content block">
                    <div class="content media is-small" v-for="(f, idx) in filesByType" :key="idx">
                      <div class="media-left">
                        <button rounded class="button is-success is-small" @click='playFile(f)'
                                v-show="f.type === 'video'">
                          <b-icon pack="fas" icon="play" size="is-small"></b-icon>
                        </button>
                        <b-tooltip :label="$t('Select this script for export')" position="is-right">
                        <button rounded class="button is-info is-small is-outlined" @click='selectScript(f)'
                          v-show="f.type === 'script'" v-bind:class="{ 'is-success': f.is_selected_script, 'is-info' :!f.is_selected_script }">
                          <b-icon pack="mdi" icon="pulse"></b-icon>
                        </button>
                        </b-tooltip>
                        <b-tooltip :label="$t('Send this script to The Handy')" position="is-right">
                        <b-button rounded size="is-small" type="is-primary" outlined @click='sendScriptToHandy(f)'
                          v-show="f.type === 'script'"
                          :loading="handyTransferingFileId === f.id"
                          :disabled="handyTransferingFileId !== 0 && handyTransferingFileId !== f.id">
                          <b-icon pack="mdi" icon="upload"></b-icon>
                        </b-button>
                        </b-tooltip>
                        <button rounded class="button is-info is-small is-outlined" disabled
                                v-show="f.type === 'hsp'">
                          <b-icon pack="mdi" icon="safety-goggles"></b-icon>
                        </button>
                        <button rounded class="button is-info is-small is-outlined" disabled
                                v-show="f.type === 'subtitles'">
                          <b-icon pack="mdi" icon="subtitles"></b-icon>
                        </button>
                      </div>
                      <div class="media-content" style="overflow-wrap: break-word;">
                        <strong>{{ f.filename }}</strong><br/>
                        <small>
                          <span class="pathDetails">{{ f.path }}</span>
                          <br/>
                          {{ prettyBytes(f.size) }}<span v-if="f.type === 'video'"> ({{ prettyBytes(f.video_bitrate, { bits: true })  }}/s)</span>,
                          <span v-if="f.type === 'video'"><span class="videosize">{{ f.video_width }}x{{ f.video_height }} {{ f.video_codec_name }}</span>, {{ f.projection }},&nbsp;</span>
                          <span v-if="f.duration > 1">{{ humanizeSeconds(f.duration) }},</span>
                          {{ format(parseISO(f.created_time), "yyyy-MM-dd") }}
                        </small>
                        <div v-if="f.type === 'script' && f.has_heatmap" class="heatmapFunscript">
                          <img :src="getHeatmapURL(f.id)"/>
                        </div>
                      </div>
                      <div class="media-right">
                        <button class="button is-dark is-small is-outlined" title="Unmatch file from scene" @click='unmatchFile(f)'>
                          <b-icon pack="fas" icon="unlink" size="is-small"></b-icon>
                        </button>&nbsp;
                        <button class="button is-danger is-small is-outlined" title="Delete file from disk" @click='removeFile(f)'>
                          <b-icon pack="fas" icon="trash" size="is-small"></b-icon>
                        </button>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item :label="`Cuepoints (${sortedCuepoints.length})`" v-if="!displayingAlternateSource">
                  <div class="block-tab-content block">
                    <div class="block" >
                      <div class="columns">
                        <div class="column is-2">
                        <b-field label="Track" width="7.25em" label-position="on-border">
                          <b-input v-model="track" width="7.25em"></b-input>
                        </b-field>
                        </div>
                        <div class="column">
                        <b-field label="Name" label-position="on-border">
                          <b-autocomplete v-model="cuepointName" :data="filteredCuepointPositionList" :open-on-focus="true"></b-autocomplete>
                        </b-field>
                        </div>
                        <div class="column is-2">
                        <b-field label="Start" label-position="on-border">
                          <b-timepicker v-model="vidPosition" rounded editable placeholder="Defaults to player position" hour-format="24" :enable-seconds="true" :max-time="maxTime" :time-formatter="timeFormatter" :time-parser="timeParser" >
                          <b-button
                            label="Current Time"
                            type="is-primary"
                            @click="vidPosition = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" />
                          </b-timepicker>
                        </b-field>
                        </div>
                        <div class="column is-2">
                          <b-field label="End" label-position="on-border">
                          <b-timepicker v-model="endTime" rounded editable placeholder="Defaults to player position" hour-format="24" :enable-seconds="true" :max-time="maxTime" :time-formatter="timeFormatter" :time-parser="timeParser" >
                          <b-button
                            label="Current Time"
                            type="is-primary"
                            @click="endTime = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" />
                          </b-timepicker>
                        </b-field>
                        </div>
                      </div>
                    </div>
                    <div>
                      <!-- :sort-multiple="sortMultiple" :sort-multiple-data="cuepointSorting" -->
                        <b-table :data="sortedCuepoints"  :narrowed=true :per-page=7 focusable striped sticky-header
                          @select="cuepointSelected">
                          <!-- paginated  pagination-position="top" :pagination-rounded=true pagination-size="is-small" -->
                          <b-table-column field="track" label="Track" width="7.25em" v-slot="props" >
                            {{ props.row.track ==null ? "" :  props.row.track }}
                          </b-table-column>
                          <b-table-column field="name" label="Name" v-slot="props"  is-small>
                            {{ props.row.name }}
                          </b-table-column>
                          <b-table-column field="time_start" label="Start" v-slot="props" width="6.5em"  >
                            {{ humanizeSeconds1DP(props.row.time_start) }}
                          </b-table-column>
                          <b-table-column field="time_end" label="End" v-slot="props" width="6.5em"  >
                            {{ props.row.time_end==null ? "" :  humanizeSeconds1DP(props.row.time_end) }}
                          </b-table-column>
                          <b-table-column field="rating" v-slot="props" width="7em"  >
                            <b-field v-if="props.row.track!=null">
                              <star-rating :key="props.row.id" v-model="props.row.rating" :rating="props.row.rating" @rating-selected="setCuepointRating(props.row)" :increment="0.5" :star-size="10" />
                              <b-icon v-if="props.row.rating>0" pack="mdi" icon="autorenew" size="is-small" @click.native="clearCuepointRating(props.row)" style="padding-left: .25em;padding-top: .5em;"/>
                            </b-field>
                          </b-table-column>
                          <b-table-column v-slot="props" width="1em" >
                            <button class="button is-danger is-outlined is-small" @click="deleteCuepoint(props.row.id)" title="Delete cuepoint">
                              <b-icon pack="fas" icon="trash" />
                            </button>
                          </b-table-column>
                        </b-table>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item label="Watch history" v-if="!displayingAlternateSource">
                  <div class="block-tab-content block">
                    <div>
                      {{ historySessionsCount }} view sessions, total duration
                      {{ humanizeSeconds(historySessionsDuration) }}
                    </div>
                    <div class="content is-small">
                      <div class="block" v-for="(session, idx) in item.history" :key="idx">
                        <strong>{{ format(parseISO(session.time_start), "yyyy-MM-dd kk:mm:ss") }} -
                          {{ humanizeSeconds(session.duration) }}</strong>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item label="Description">
                  <div class="block-tab-content block">
                    <b-message>
                      {{ item.synopsis }}
                    </b-message>
                  </div>
                </b-tab-item>
                <b-tab-item v-if="this.$store.state.optionsAdvanced.advanced.showSceneSearchField && !displayingAlternateSource" label="Search fields">
                  <div class="block-tab-content block">
                    <div class="content is-small">
                      <div class="block" v-for="(field, idx) in searchfields" :key="idx">
                        <strong>{{ field.fieldName }} - </strong> {{ field.fieldValue }}
                      </div>
                    </div>
                  </div>
                </b-tab-item>

              </b-tabs>
            </div>

          </div>
        </div>
      </section>
      <div class="scene-id">
        {{ item.scene_id }}
        <span  v-if="this.$store.state.optionsAdvanced.advanced.showInternalSceneId">{{ $t('Internal ID') }}: {{item.id}}</span>
        <a v-if="this.$store.state.optionsAdvanced.advanced.showHSPApiLink" :href="`/heresphere/${item.id}`" target="_blank" rel="noreferrer" style="margin-left:0.5em">
          <img src="/ui/icons/heresphere_24.png" style="height:15px;"/>
        </a>
      </div>
    </div>
    <button v-if="!embedded" class="modal-close is-large" aria-label="close" @click="close()"></button>
    <button v-else-if="!isStandalonePage" class="button is-small is-light scene-details-inline-close" @click="close()">Close</button>
    <button v-else class="button is-small is-light scene-details-page-back" @click="close()">Back to scenes</button>
    <a class="prev" @click="prevScene" v-if="$store.getters['sceneList/prevScene'](item) !== null && !displayingAlternateSource"
       title="Keyboard shortcut: O">&#10094;</a>
    <a class="next" @click="nextScene" v-if="$store.getters['sceneList/nextScene'](item) !== null && !displayingAlternateSource"
       title="Keyboard shortcut: P">&#10095;</a>
  </div>
</template>

<script>
import ky from 'ky'
import videojs from 'video.js'
import 'videojs-vr/dist/videojs-vr.min.js'
import { format, formatDistance, parseISO } from 'date-fns'
import prettyBytes from 'pretty-bytes'
import VueLoadImage from 'vue-load-image'
import GlobalEvents from 'vue-global-events'
import StarRating from 'vue-star-rating'
import FavouriteButton from '../../components/FavouriteButton'
import LinkStashdbButton from '../../components/LinkStashdbButton'
import WatchlistButton from '../../components/WatchlistButton'
import WishlistButton from '../../components/WishlistButton'
import WatchedButton from '../../components/WatchedButton'
import EditButton from '../../components/EditButton'
import RefreshButton from '../../components/RefreshButton'
import RescrapeButton from '../../components/RescrapeButton'
import TrailerlistButton from '../../components/TrailerlistButton'
import HiddenButton from '../../components/HiddenButton'
import { loadHandyConfig, selectHandyScriptFile, sendHandyTransfer, stopHandyScript } from '../../lib/handy'

export default {
  name: 'Details',
  props: {
    embedded: {
      type: Boolean,
      default: false
    }
  },
  components: { VueLoadImage, GlobalEvents, StarRating, WatchlistButton, FavouriteButton, LinkStashdbButton, WishlistButton, WatchedButton, EditButton, RefreshButton, RescrapeButton, TrailerlistButton, HiddenButton },
  data () {
    return {
      index: 1,
      activeTab: 0,
      activeMedia: 0,
      player: {},
      tagAct: '',
      cuepointName: '',
      cuepointRating: 0,
      cuepointPositionTags: ['', 'standing', 'sitting', 'laying', 'kneeling'],
      cuepointActTags: ['', 'handjob', 'blowjob', 'doggy', 'cowgirl', 'revcowgirl', 'missionary', 'titfuck', 'anal', 'cumshot', '69', 'facesit'],
      carouselSlide: 0,
      vidPosition: null,
      skipForwardIntervals: [5, 10, 30, 60, 120, 300],
      skipBackIntervals: [-300, -120, -60, -30, -10, -5],
      lastSkipFowardInterval: 5,
      lastSkipBackInterval: -5,
      currentCuepointId: 0,
      maxTime: new Date(0, 0, 0, 5, 0, 0),
      cuepointSorting: [{ field: "is_hsp", order: "asc" },{ field: "time_start", order: "desc" }, {field: "track", order: "desc"}, {field: "time_end", order: "desc"}],
      trackInput: '',
      track: null,
      endTime: null,
      sortMultiple: true,
      castimages: [],
      searchfields: [],
      alternateSources: [],
      waitingForQuickFind: false,
      handyReady: false,
      handyScriptFileId: 0,
      handySetupPromise: null,
      handyClosed: false,
      handyTransferingFileId: 0,
    }
  },
  computed: {
    containerClass () {
      if (!this.embedded) return 'modal is-active'
      return this.isStandalonePage ? 'scene-details-page' : 'scene-details-inline'
    },
    layoutClass () {
      return this.isStandalonePage ? 'scene-details-page-layout' : 'columns'
    },
    mediaColumnClass () {
      return this.isStandalonePage ? 'scene-details-page-media' : 'column is-half'
    },
    infoColumnClass () {
      return this.isStandalonePage ? 'scene-details-page-info' : 'column is-half'
    },
    item () {
      const item = this.embedded
        ? this.$store.state.overlay.inlineDetails.scene
        : this.$store.state.overlay.details.scene
      if (!item) {
        return {
          cast: [],
          tags: [],
          file: [],
          history: [],
          images: '[]',
          release_date: '0001-01-01T00:00:00Z',
          scene_id: 0
        }
      }
      if (this.$store.state.optionsWeb.web.tagSort === 'alphabetically') {
        item.tags.sort((a, b) => a.name < b.name ? -1 : 1)
      }
      let releasedate = parseISO(item.release_date)
      let imgs = item.cast.map((actor) => {
        let birthdate = parseISO(actor.birth_date)
        let label = actor.name
        if (birthdate.getFullYear() > 0) {
          let age = releasedate.getFullYear() - birthdate.getFullYear()
          if ((releasedate.getMonth() < birthdate.getMonth()) || (releasedate.getMonth() == birthdate.getMonth() && releasedate.getDate() < birthdate.getDate())) {
            age -= 1
          }
          label += `, ${age} in scene`
        }
        let img = actor.image_url
        if (img == "" ){
          img = "blank"  // forces an error image to load, blank won't display an image
        }
        if (actor.name.startsWith("aka:")) {
          img = ""
        }
        return {src: img, visible: false, actor_name: actor.name, actor_label: label, actor_id: actor.id};
      });

      this.castimages =  imgs.filter((img) => {
        return img.src !== '';
        });
      this.getSearchFields(item.id)
      return item
    },
    // Properties for gallery
    images () {
      if (this.item.images=="null") {
        return "[]"
      }
      return JSON.parse(this.item.images).filter(im => im && im.url)
    },
    // Tab: cuepoints
    sortedCuepoints () {
      if (this.item.cuepoints !== null) {
        for (let i = 0; i < this.item.cuepoints.length; i++) {
          this.item.cuepoints[i].is_hsp = this.item.cuepoints[i].track == null ? 0 : 1
        }
        let x=this.item.cuepoints.slice().sort((a, b) => (a.time_start > b.time_start) ? 1 : -1 || (a.is_hsp >b.is_hsp) ? 1 : -1 )
        x=this.item.cuepoints.slice().sort((a,b) => {
          let compare = (a.is_hsp<b.is_hsp) ? -1 : (a.is_hsp>b.is_hsp) ? 1 : 0
          if (compare!=0) {
            return compare
          }
          compare = (a.time_start<b.time_start) ? -1 : (a.time_start>b.time_start) ? 1 : 0
          if (compare!=0) {
            return compare
          }
          compare = (a.track<b.track) ? -1 : (a.track>b.track) ? 1 : 0
          if (compare!=0) {
            return compare
          }
          return  (a.time_end<b.time_end) ? -1 : (a.time_end>b.time_end) ? 1 : 0
        })
        return x
      }
      return []
    },
    // Tab: files
    fileCount () {
      if (this.item.file !== null) {
        return this.item.file.length
      }
      return 0
    },
    bestVideoFile () {
      const videoFiles = (this.item.file || []).filter(f => f.type === 'video')
      if (videoFiles.length === 0) {
        return null
      }
      return videoFiles.slice().sort(this.compareVideoQuality)[0]
    },
    filesByType () {
      if (this.item.file !== null) {
        return this.item.file.slice().sort((a, b) => {
          if (a.type === 'video' && b.type === 'video') {
            return this.compareVideoQuality(a, b)
          }
          if (a.type === 'video') return -1
          if (b.type === 'video') return 1
          return 0
        })
      }
      return []
    },
    // Tab: history
    historySessionsCount () {
      if (this.item.history !== null) {
        return this.item.history.length
      }
      return 0
    },
    historySessionsDuration () {
      if (this.item.history !== null) {
        let total = 0
        this.item.history.slice().map(i => {
          total = total + i.duration
          return 0
        })
        return total
      }
      return 0
    },
    showEdit () {
      return this.$store.state.overlay.edit.show
    },
    filteredCuepointPositionList () {
      // filter the list of positions based on what has been entered so far
      let list=this.cuepointActTags.concat(this.cuepointPositionTags)
      return list.filter((option) => {
        return option
          .toString()
          .toLowerCase()
          .trim()
          .indexOf(this.cuepointName.toString().toLowerCase()) >= 0
      })
    },
    displayingAlternateSource () {
      // displayingAlternateSource indicates we aren't displaying a real xbvr scene from the scenes table,
      //  so functions like watchlist, ratings, etc don't apply
      // we are displaying scene data serialized and saved in the external_references table
      const overlayState = this.embedded
        ? this.$store.state.overlay.inlineDetails
        : this.$store.state.overlay.details
      if (overlayState.altsrc != null) return true
      return false
    },
    async getAlternateSceneSources() {
      this.alternateSources = [];
      if (this.displayingAlternateSource) return 0
      try {
        const response = await ky.get('/api/scene/alternate_source/' + this.item.id).json();
        if (response==null){
          return 0
        }
        response.forEach(altsrc => {
          if (altsrc.external_source.startsWith("alternate scene ")) {
            this.alternateSources.push(altsrc)
          }
        });
        return this.alternateSources.length;
      } catch (error) {        
        return 0; // Return 0 or handle error as needed
      }
    },
    changeDetailsTab() {      
      return this.$store.state.overlay.changeDetailsTab
    },
    quickFindOverlayState() {
      return this.$store.state.overlay.quickFind.show
    },
    showOpenInNewWindow () {
      return this.$store.state.optionsWeb.web.showOpenInNewWindow
    },
    isStandalonePage () {
      return this.embedded && this.$route.name === 'scene'
    },
    playerAspectRatio () {
      return this.isStandalonePage ? '16:9' : '1:1'
    },
    alternateSourcesWithTitles() {
      return this.alternateSources.map(altsrc => {
        const extdata = JSON.parse(altsrc.external_data);
        return {
          ...altsrc,
          title: extdata.scene?.title || 'No Title'
        };
      });
    }
  },
  mounted () {
    if (this.isStandalonePage && !this.displayingAlternateSource) {
      this.activeMedia = 1
    }
    this.setupPlayer()
    this.handySetupPromise = this.setupHandy()

    // load default cuepoint actions & positions from kv entry in the db
    ky.get('/api/options/cuepoints').json().then(data => {
      this.cuepointActTags = data.actions
      this.cuepointPositionTags = data.positions
      this.cuepointActTags.unshift("")
      this.cuepointPositionTags.unshift("")
      })    
},
watch:{
  quickFindOverlayState(newVal, oldVal){
    if (newVal == true) {
      return
    }
    if (this.waitingForQuickFind){
      this.waitingForQuickFind = false
      if (this.$store.state.overlay.quickFind.selectedScene != null && this.$store.state.overlay.quickFind.selectedScene.id > 0) {
        this.$buefy.dialog.confirm({
          title: 'Relink scene',
          message: `Do you wish to link this scene to <strong>${this.$store.state.overlay.quickFind.selectedScene.title}</strong>`,
          type: 'is-info is-wide',
          hasIcon: true,
          id: 'heh',
          onConfirm: () => {
            this.handleRelinkExtRef()
          }
        })
      }
    }
  },
  changeDetailsTab(newVal, oldVal){
    if (newVal == -1 ) {
      return
    }
    this.activeTab = newVal
    this.$store.commit('overlay/changeDetailsTab', { tab: -1 })
  },
  activeMedia(newVal, oldVal) {
    // Auto-load first video when Player tab is opened (without auto-playing)
    // The webUI video player doesn't work for some users without autoloading
    if (newVal === 1 && !this.displayingAlternateSource) {
      if (this.bestVideoFile) {
        this.activeMedia = 1
        this.updatePlayer('/api/dms/file/' + this.bestVideoFile.id + '?dnt=true', 'NONE')
      }
    }
  }
},
  methods: {
    compareVideoQuality (a, b) {
      const numericFields = [
        'video_height',
        'video_bitrate',
        'video_width',
        'size'
      ]

      for (const field of numericFields) {
        const av = Number(a[field] || 0)
        const bv = Number(b[field] || 0)
        if (av !== bv) {
          return bv - av
        }
      }

      const aTime = Date.parse(a.created_time || '') || 0
      const bTime = Date.parse(b.created_time || '') || 0
      if (aTime !== bTime) {
        return bTime - aTime
      }

      return (a.filename || '').localeCompare(b.filename || '')
    },
    showSceneOverlay (payload) {
      if (this.isStandalonePage && payload && payload.scene && payload.scene.id) {
        this.$store.commit('overlay/showInlineDetails', payload)
        if (String(this.$route.params.id) !== String(payload.scene.id)) {
          this.$router.push({
            name: 'scene',
            params: { id: String(payload.scene.id) },
            query: this.$route.query
          })
        }
        return
      }
      if (this.embedded) {
        this.$store.commit('overlay/showInlineDetails', payload)
      } else {
        this.$store.commit('overlay/showDetails', payload)
      }
    },
    setupPlayer () {
      this.player = videojs(this.$refs.player, {
        aspectRatio: this.playerAspectRatio,
        fluid: true,
        loop: true
      })

      this.player.hotkeys({
        alwaysCaptureHotkeys: true,
        volumeStep: 0.1,
        seekStep: 5,
        enableModifiersForNumbers: false,
        enableVolumeScroll: false,
        customKeys: {
          closeModal: {
            key: function (event) {
              return event.which === 27
            },
            handler: (player, options, event) => {
              this.close()
            }
          },
          zoomIn: {
            handler: (player, options, event) => {
              this.zoomHandler(true)
            }
          },
          zoomOut: {
            handler: (player, options, event) => {
              this.zoomHandler(false)
            }
          }
        }
      })

      const videoElement = this.player.el();
      videoElement.addEventListener('wheel', this.zoomHandlerWeb.bind(this))
      this.player.on('play', () => {
        this.syncHandyPlayback()
      })
      this.player.on('pause', () => {
        this.stopHandyPlayback()
      })
      this.player.on('ended', () => {
        this.stopHandyPlayback()
      })
      this.player.on('seeked', () => {
        this.syncHandyPlayback()
      })
    },

    zoomHandlerWeb(event) {
      if (!(event.ctrlKey || event.metaKey)) {
        return
      }
      event.preventDefault();
      this.zoomHandler(event.deltaY < 0)
    },

    zoomHandler(isZoomingIn) {
      const vr = this.player.vr()
      const minFov = 30
      const maxFov = 130
      let fov = vr.camera.fov + (isZoomingIn ? -1 : 1)

      if (fov < minFov) {
        fov = minFov
      }

      if (fov > maxFov) {
        fov = maxFov
      }

      vr.camera.fov = fov;
      vr.camera.updateProjectionMatrix()
    },
    async setupHandy () {
      try {
        if (this.displayingAlternateSource) {
          return
        }

        const handyConfig = await loadHandyConfig()
        if (this.handyClosed || !handyConfig.enabled || !handyConfig.connection_key) {
          return
        }

        const scriptFile = selectHandyScriptFile(this.item)
        if (!scriptFile) {
          console.info('[Handy] no funscript found for scene', { sceneId: this.item?.id })
          return
        }

        await this.transferHandyScript(scriptFile, {
          context: 'scene',
          handyConfig,
          syncPlayback: true
        })
      } catch (error) {
        console.warn('[Handy] scene player setup failed', error)
      }
    },
    async transferHandyScript (scriptFile, options = {}) {
      if (!scriptFile || this.handyClosed) {
        return false
      }

      const context = options.context || 'manual'
      const handyConfig = options.handyConfig || await loadHandyConfig()
      if (!handyConfig.enabled || !handyConfig.connection_key) {
        if (context === 'manual') {
          this.$buefy.toast.open({
            message: 'The Handy is disabled or missing a connection key.',
            type: 'is-warning',
            duration: 3500
          })
        }
        return false
      }

      if (this.handyTransferingFileId === scriptFile.id) {
        return false
      }

      this.handyTransferingFileId = scriptFile.id
      try {
        console.info('[Handy] preparing script upload', { context, sceneId: this.item?.id, scriptId: scriptFile.id })
        const play = Boolean(options.play)
        const startTimeMs = typeof options.startTimeMs === 'number'
          ? options.startTimeMs
          : 0
        const result = await sendHandyTransfer(this.item.id, scriptFile.id, startTimeMs, play)
        if (this.handyClosed) {
          return false
        }

        this.handyScriptFileId = scriptFile.id
        this.handyReady = true
        console.info('[Handy] script uploaded', { context, sceneId: this.item?.id, scriptId: scriptFile.id, result })

        if (options.syncPlayback && !this.player.paused()) {
          await this.syncHandyPlayback()
        }

        if (context === 'manual') {
          this.$buefy.toast.open({
            message: `Sent ${scriptFile.filename} to The Handy.`,
            type: 'is-primary',
            duration: 3000
          })
        }
        return true
      } catch (error) {
        console.warn('[Handy] script transfer failed', { context, sceneId: this.item?.id, scriptId: scriptFile.id, error })
        if (context === 'manual') {
          this.$buefy.toast.open({
            message: `Failed to send ${scriptFile.filename} to The Handy.`,
            type: 'is-danger',
            duration: 3500
          })
        }
        return false
      } finally {
        if (this.handyTransferingFileId === scriptFile.id) {
          this.handyTransferingFileId = 0
        }
      }
    },
    async sendScriptToHandy (file) {
      if (!file || file.type !== 'script') {
        return
      }

      if (this.handyTransferingFileId === file.id) {
        return
      }

      this.handyTransferingFileId = file.id
      try {
        const play = Boolean(this.player && typeof this.player.paused === 'function' && !this.player.paused())
        const startTimeMs = this.player && typeof this.player.currentTime === 'function'
          ? Math.round(this.player.currentTime() * 1000)
          : 0

        const result = await sendHandyTransfer(this.item.id, file.id, startTimeMs, play)

        this.handyScriptFileId = file.id
        this.handyReady = true
        console.info('[Handy] manual transfer completed', result)
        this.$buefy.toast.open({
          message: `Sent ${file.filename} to The Handy.`,
          type: 'is-primary',
          duration: 3000
        })
      } catch (error) {
        console.warn('[Handy] manual transfer failed', { sceneId: this.item?.id, scriptId: file.id, error })
        this.$buefy.toast.open({
          message: `Failed to send ${file.filename} to The Handy.`,
          type: 'is-danger',
          duration: 3500
        })
      } finally {
        if (this.handyTransferingFileId === file.id) {
          this.handyTransferingFileId = 0
        }
      }
    },
    async syncHandyPlayback () {
      if (!this.handyReady || this.handyClosed || this.player.paused() || !this.handyScriptFileId) {
        return
      }

      if (!this.item || !this.item.id) {
        return
      }

      await sendHandyTransfer(this.item.id, this.handyScriptFileId, Math.round(this.player.currentTime() * 1000), true)
    },
    async stopHandyPlayback () {
      if (!this.handyReady || this.handyClosed) {
        return
      }

      try {
        console.info('[Handy] stopping script for scene player', { sceneId: this.item?.id })
        await stopHandyScript()
        console.info('[Handy] stopped script for scene player', { sceneId: this.item?.id })
      } catch (error) {
        console.warn('[Handy] scene player stop failed', error)
      }
    },
    updatePlayer (src, projection) {
      this.player.reset()
      /* const vr = */ this.player.vr({
        projection: projection,
        forceCardboard: false
      })

      this.player.on('loadedmetadata', function () {
        // vr.camera.position.set(-1, 0, 2);
      })

      if (src) {
        this.player.src({ src: src, type: 'video/mp4' })
      }
      this.player.poster(this.getImageURL(this.item.cover_url, ''))
    },
    showCastScenes (actor) {
      this.$store.state.sceneList.filters.cast = actor
      this.$store.state.sceneList.filters.sites = []
      this.$store.state.sceneList.filters.tags = []
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    getCastScenesUrl(actor) {
      let newfilters = Object.assign({}, this.$store.state.sceneList.filters);
      newfilters.cast = actor;       
      newfilters.sites = []
      newfilters.tags = []
      newfilters.attributes = []
      return this.$router.resolve({
        name: 'scenes',
        query: { q: Buffer.from(JSON.stringify(newfilters)).toString('base64') }
      }).href
    },
    showTagScenes (tag) {
      this.$store.state.sceneList.filters.cast = []
      this.$store.state.sceneList.filters.sites = []
      this.$store.state.sceneList.filters.tags = tag
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    getTagScenesUrl(tag) {
      let newfilters = Object.assign({}, this.$store.state.sceneList.filters);      
      newfilters.tags = tag;       
      newfilters.cast = []       
      newfilters.sites = []
      newfilters.attributes = []
      return this.$router.resolve({
        name: 'scenes',
        query: { q: Buffer.from(JSON.stringify(newfilters)).toString('base64') }
      }).href
    },
    showSiteScenes (site) {
      this.$store.state.sceneList.filters.cast = []
      this.$store.state.sceneList.filters.sites = site
      this.$store.state.sceneList.filters.tags = []
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    getSiteScenesUrl(site) {
      let newfilters = Object.assign({}, this.$store.state.sceneList.filters);
      newfilters.sites = site;       
      newfilters.cast = []       
      newfilters.tags = []
      newfilters.attributes = []
      return this.$router.resolve({
        name: 'scenes',
        query: { q: Buffer.from(JSON.stringify(newfilters)).toString('base64') }
      }).href
    },
    showActorDetail (actor_id) {
      ky.get('/api/actor/'+actor_id).json().then(data => {
        if (data.id != 0){
          this.$store.commit('overlay/showActorDetails', { actor: data })
          this.close()
        }
      })
    },
    playPreview () {
      this.activeMedia = 1
      this.updatePlayer('/api/dms/preview/' + this.item.scene_id, 'NONE')
      this.player.play()
    },
    playFile (file) {
      this.activeMedia = 1
      this.updatePlayer('/api/dms/file/' + file.id + '?dnt=true', 'NONE')
      this.player.play()
    },
    unmatchFile (file) {
      this.$buefy.dialog.confirm({
        title: 'Unmatch file',
        message: `You're about to unmatch the file <strong>${file.filename}</strong> from this scene. Afterwards, it can be matched again to this or any other scene.`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {
          ky.post(`/api/files/unmatch`, {json:{file_id: file.id}}).json().then(data => {
            this.$store.commit('sceneList/updateScene', data)
            this.$store.dispatch('sceneList/load', { offset: 0 })
            this.showSceneOverlay({ scene: data })
          })
        }
      })
    },
    removeFile (file) {
      this.$buefy.dialog.confirm({
        title: 'Remove file',
        message: `You're about to remove file <strong>${file.filename}</strong> from <strong>disk</strong>.`,
        type: 'is-danger',
        hasIcon: true,
        onConfirm: () => {
          ky.delete(`/api/files/file/${file.id}`).json().then(data => {
            this.$store.commit('sceneList/updateScene', data)
            this.$store.dispatch('sceneList/load', { offset: 0 })
            this.showSceneOverlay({ scene: data })
          })
        }
      })
    },
    selectScript (file) {
      ky.post(`/api/scene/selectscript/${this.item.id}`, {
        json: {
          file_id: file.id,
        }
      }).json().then(data => {
          this.showSceneOverlay({ scene: data })
      })
    },
    getImageURL (u, size) {
      if (u==undefined) {
        return u
      }
      try {
        if (u.startsWith('http')) {
          if (u.search("%") == -1) {
            return '/img/' + size + '/' + encodeURI(u)
          } else {
            return '/img/' + size + '/' + encodeURI(decodeURI(u))
          } 
          return u
        }
      } catch {
        return u
      }
      return u
    },
    getIndicatorURL (idx) {
      if (this.images[idx] !== undefined) {
        return this.getImageURL(this.images[idx].url, 'x40')
      } else {
        return '/ui/images/blank.png'
      }
    },
    getHeatmapURL (fileId) {
      return `/api/dms/heatmap/${fileId}`
    },
    playCuepoint (cuepoint) {
      // populate the cuepoint edit fields
      this.vidPosition = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_start*1000)
      this.endTime = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_end*1000)
      this.currentCuepointId = cuepoint.id
      this.cuepointRating = cuepoint.rating
      if (cuepoint.name.indexOf('-') > 0) {
        this.cuepointName = cuepoint.name.substr(0, cuepoint.name.indexOf('-'))
        this.tagAct = cuepoint.name.substr(cuepoint.name.indexOf('-') + 1)
      } else {
        this.tagAct = cuepoint.name
        this.cuepointName = ''
      }
      // now mow the player position
      this.player.currentTime(cuepoint.time_start)
      this.player.play()
    },
    updateCuepoint (editCuepoint) {
      if (this.disableSaveButtons()) return
      // if edit choosen, delete existing cuepoint before add
      if (editCuepoint && this.currentCuepointId > 0) {
        this.deleteCuepoint(this.currentCuepointId)
      }
      let name =  this.cuepointName
      let pos = this.player.currentTime()
      let endpos=null
      this.track=parseInt(this.track)
      if (this.vidPosition != null) {
        pos = (this.vidPosition.getMilliseconds() / 1000) + this.vidPosition.getSeconds() + (this.vidPosition.getMinutes() * 60) + (this.vidPosition.getHours() * 60 * 60)
      }
      if (this.endTime != null) {
        endpos = (this.endTime.getMilliseconds() / 1000) + this.endTime.getSeconds() + (this.endTime.getMinutes() * 60) + (this.endTime.getHours() * 60 * 60)
      }
      this.currentCuepointId = 0

      ky.post(`/api/scene/${this.item.id}/cuepoint`, {
        json: {
          track: this.track,
          name: name,
          time_start: pos,
          time_end: endpos,
          rating: this.cuepointRating
        }
      }).json().then(data => {
        this.vidPosition = null
        this.endTime = null
        this.cuepointName=''
        this.track = null
        this.$store.commit('sceneList/updateScene', data)
        this.showSceneOverlay({ scene: data })
      })
    },
    deleteCuepoint (cuepointid) {
      ky.delete(`/api/scene/${this.item.id}/cuepoint/${cuepointid}`)
        .json().then(data => {
          this.$store.commit('sceneList/updateScene', data)
          this.showSceneOverlay({ scene: data })
        })
    },
    async close () {
      if (this.handyReady) {
        try {
          await stopHandyScript()
        } catch (error) {
          console.warn('[Handy] scene player close stop failed', error)
        }
      }
      this.handyClosed = true
      if (this.player && !this.player.paused()) {
        this.player.pause()
      }
      if (!this.displayingAlternateSource) this.player.dispose()
      if (this.isStandalonePage) {
        this.$store.commit('overlay/hideInlineDetails')
        this.$router.push({
          name: 'scenes',
          query: this.$route.query
        })
      } else if (this.embedded) {
        this.$store.commit('overlay/hideInlineDetails')
      } else {
        this.$store.commit('overlay/hideDetails')
      }
    },
    humanizeSeconds (seconds) {
      return new Date(seconds * 1000).toISOString().substr(11, 8)
    },
    humanizeSeconds1DP (seconds) {
      return new Date(seconds * 1000).toISOString().substr(11, 10)
    },
    setRating (val) {
      ky.post(`/api/scene/rate/${this.item.id}`, { json: { rating: val } })

      const updatedScene = Object.assign({}, this.item)
      updatedScene.star_rating = val
      this.item.star_rating = val
      this.$store.commit('sceneList/updateScene', updatedScene)
    },
    nextScene () {
      const data = this.$store.getters['sceneList/nextScene'](this.item)
      if (data !== null && !this.displayingAlternateSource) {
        this.showSceneOverlay({ scene: data })
        this.activeMedia = this.isStandalonePage ? 1 : 0
        this.carouselSlide = 0
        this.updatePlayer(undefined, '180')
      }
    },
    prevScene () {
      const data = this.$store.getters['sceneList/prevScene'](this.item)
      if (data !== null && !this.displayingAlternateSource) {
        this.showSceneOverlay({ scene: data })
        this.activeMedia = this.isStandalonePage ? 1 : 0
        this.carouselSlide = 0
        this.updatePlayer(undefined, '180')
      }
    },
    playerStepBack (interval) {
      const wasPlaying = !this.player.paused()
      if (wasPlaying) {
        this.player.pause()
      }
      let seekTime = this.player.currentTime() + interval
      if (seekTime <= 0) {
        seekTime = 0
      }
      this.player.currentTime(seekTime)
      if (wasPlaying) {
        this.player.play()
      }
      this.lastSkipBackInterval = interval
    },
    playerStepForward (interval) {
      const duration = this.player.duration()
      const wasPlaying = !this.player.paused()
      if (wasPlaying) {
        this.player.pause()
      }
      let seekTime = this.player.currentTime() + interval
      if (seekTime >= duration) {
        seekTime = wasPlaying ? duration - 0.001 : duration
      }
      this.player.currentTime(seekTime)
      if (wasPlaying) {
        this.player.play()
      }
      this.lastSkipFowardInterval = interval
    },
    setCuepointName (param) {
      if (this.activeTab === 1) {
        if (this.cuepointName=='') {
          this.cuepointName = param.toString()
        }else{
          this.cuepointName = this.cuepointName+'-'+param.toString()
        }
      }
    },
    toggleGallery () {
      if (this.activeMedia == 0) {
        this.activeMedia = 1
      } else {
        this.activeMedia = 0
        }
    },
    handleLeftArrow () {
      if (this.activeMedia === 0)
      {
        this.carouselSlide = this.carouselSlide - 1
      } else {
        this.playerStepBack(this.lastSkipBackInterval)
      }
    },
    handleRightArrow () {
      if (this.activeMedia === 0)
      {
        this.carouselSlide = this.carouselSlide + 1
      } else {
        this.playerStepForward(this.lastSkipFowardInterval)
      }
    },
    scrollToActiveIndicator (value) {
      const indicators = document.querySelector('.carousel-indicator')
      const active = indicators.children[value]
      indicators.scrollTo({
        top: 0,
        left: active.offsetLeft + active.offsetWidth / 2 - indicators.offsetWidth / 2,
        behavior: 'smooth'
      })
    },
    timeFormatter(time) {
       return new Intl.DateTimeFormat('en', { hourCycle: 'h23', hour: "2-digit", minute: "2-digit", second: "2-digit", fractionalSecondDigits: 1 }).format(time)
    },
    timeParser(inputString) {
      let items = inputString.split(":")
      return new Date(0, 0, 0, items[0],items[1], 0, items[2]*1000)
    },
    cuepointSelected(cuepoint) {
      // populate the cuepoint edit fields
      this.vidPosition = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_start*1000)
      this.endTime = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_end*1000)
      this.currentCuepointId = cuepoint.id
      this.cuepointName = cuepoint.name
      this.track=cuepoint.track
      this.cuepointRating=cuepoint.rating
      // now mow the player position
      this.player.currentTime(cuepoint.time_start)
      this.player.play()
    },
    disableSaveButtons() {
      if (this.track!=null && this.track!="" && (isNaN(this.endTime) || this.endTime==null)) return true
      if ((this.track==null || this.track==="") && !isNaN(this.endTime) && this.endTime!=null) return true
      return false
    },
    disableSaveMsg() {
      if (this.track!=null && this.track!="" && (isNaN(this.endTime) || this.endTime==null)) return "Specify a End Time"
      if ((this.track==null || this.track==="") && !isNaN(this.endTime) && this.endTime!=null) return "End Time is only valid for HSP Cuepoints"
      return ""
    },
    setCuepointRating (row) {
      this.cuepointSelected(row)
      this.updateCuepoint(true)
    },
    clearCuepointRating (row) {
      row.rating=0
      this.cuepointSelected(row)
      this.updateCuepoint(true)
    },
    showTooltip(idx) {
      this.castimages[idx].visible = true;
    },
    hideTooltip(idx) {
      this.castimages[idx].visible = false;
    },
    getSearchFields(id) {
      // load search fields
      this.searchfields = []      
      if (this.$store.state.optionsAdvanced.advanced.showSceneSearchField && !this.displayingAlternateSource) {
        ky.get('/api/scene/searchfields', {
          searchParams: {
            q: id
          },
          }).json().then(data => {
            this.searchfields = data
          })
      }
    },
    showExtRefScene (altsrc) {      
      const extdata = JSON.parse(altsrc.external_data);      
      if (extdata.scene.cast == null) 
      {
        extdata.scene.cast = []
      }
      this.showSceneOverlay({ scene: extdata.scene, altsrc: altsrc, prevscene: this.item, query_for_altsrc: extdata.query })
      this.activeTab = 0      
    },
    searchAlternateSourceScene() {
      // search for a new scene to link to the alternate source scene
      const overlayState = this.embedded ? this.$store.state.overlay.inlineDetails : this.$store.state.overlay.details
      const q = overlayState.query_for_altsrc == "" ? this.item.title : overlayState.query_for_altsrc
      this.$store.commit('overlay/showQuickFind', { searchString:  q, displaySelectedScene: false })
      this.waitingForQuickFind = true
    }, 
    async handleRelinkExtRef() {
      const overlayState = this.embedded ? this.$store.state.overlay.inlineDetails : this.$store.state.overlay.details
      const response = await ky.post(`/api/extref/edit_link`, {
        json: {
          external_source: overlayState.altsrc.external_source,
          external_id: overlayState.altsrc.external_id,
          internal_table: "scenes",
          internal_db_id: this.$store.state.overlay.quickFind.selectedScene.id,
          internal_name_id: this.$store.state.overlay.quickFind.selectedScene.scene_id,
          match_type: 99999
        }
      });
      if (response.status === 200) {
        overlayState.prevscene = this.$store.state.overlay.quickFind.selectedScene;
        this.$buefy.toast.open({ message: `The scene was sucessfully relinked to a new Scene`, type: 'is-primary', duration: 3000 });
      }
    },
    async scrapeScene() {
      const overlayState = this.embedded ? this.$store.state.overlay.inlineDetails : this.$store.state.overlay.details
      this.$buefy.dialog.confirm({
        title: 'Scrape & Create Scene',
        message: `Do you wish to create a seperate XBVR scene from this linked scene <strong>${overlayState.altsrc.url}</strong>`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {
          const url = overlayState.altsrc.url
          overlayState.altsrc = null
          if (this.embedded) {
            this.$store.commit('overlay/hideInlineDetails')
          } else {
            this.$store.commit('overlay/hideDetails')
          }
          // call the options screen passing the url in state   
          this.$store.commit('optionsSceneCreate/setScrapeScene', url )
          this.$store.commit('optionsSceneCreate/showSceneCreate', true )
          this.$router.push({ path: '/options'})
        }
      })

    },
    async refreshExtRef() {
      this.$buefy.dialog.confirm({
        title: 'Continue?',
        message: `This will remove the scene, rescrape the site to relink it to an XBVR scene`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {          
          this.handleRefreshExtRef()
        }
      })
    },
    async handleRefreshExtRef() {
      const overlayState = this.embedded ? this.$store.state.overlay.inlineDetails : this.$store.state.overlay.details
      const response = await ky.delete(`/api/extref/delete_extref`, {
        json: {
          external_source: overlayState.altsrc.external_source,
          external_id: overlayState.altsrc.external_id,
        }
      });
      if (response.status === 200) {
        overlayState.prevscene = this.$store.state.overlay.quickFind.selectedScene;
        this.$buefy.toast.open({ message: `The scene was removed, ready to rescan`, type: 'is-primary', duration: 3000 });
      }
    },
    flagExtRefDeleted() {
      let confirmed = false
      this.$buefy.dialog.confirm({
        title: 'Continue?',
        message: `This will unlink the scene and prevent it from relinking to any scene. This cannot be undone`,
        type: 'is-danger is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {          
          this.handleFlagExtRefDeleted()
        },
      })    
    },    
    async handleFlagExtRefDeleted() {
      const overlayState = this.embedded ? this.$store.state.overlay.inlineDetails : this.$store.state.overlay.details
      const response = await ky.post(`/api/extref/edit_link`, {
        json: {
          external_source: overlayState.altsrc.external_source,
          external_id: overlayState.altsrc.external_id,
          internal_table: "scenes",
          internal_db_id: 0,
          internal_name_id: "deleted",
          match_type: -1
        }
      });
      if (response.status === 200) {
        overlayState.prevscene = this.$store.state.overlay.quickFind.selectedScene;
        this.$buefy.toast.open({ message: `The scene was unlinked and will not be relinked to any scene`, type: 'is-primary', duration: 3000 });
      }
    },    
    format,
    parseISO,
    prettyBytes,
    formatDistance
  }
}
</script>

<style lang="less" scoped>
.bbox {
  flex: 1 0 calc(25%);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 0;
  line-height: 0;
}

.is-1by1 {
  padding-top: calc(100% - 40px - 1em) !important;
}

.video-js {
  margin: 0 auto;
}

.scene-details-inline,
.scene-details-page {
  position: relative;
  background: #fff;
  padding: 1rem;
}

.scene-details-page {
  max-width: 1400px;
  margin: 0 auto;
}

.scene-details-inline {
  border-radius: 12px;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.16);
}

.scene-details-inline .modal-card,
.scene-details-page .modal-card {
  width: 100%;
  max-width: none;
  margin: 0;
}

.scene-details-page .modal-card {
  display: block;
  max-height: none;
  overflow: visible;
  height: auto;
}

.scene-details-inline .modal-card-body,
.scene-details-page .modal-card-body {
  padding: 1rem;
}

.scene-details-page .modal-card-body {
  display: block;
  overflow: visible;
  max-height: none;
  height: auto;
  flex: none;
}

.scene-details-inline-close {
  position: absolute;
  top: 0.75rem;
  right: 0.75rem;
  z-index: 2;
}

.scene-details-page-back {
  margin: 0 0 1rem 0;
}

.scene-details-page-layout {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.scene-details-page-media {
  width: 100%;
}

.scene-details-page-info {
  display: block;
  width: 100%;
  padding-top: 0;
  border-top: 1px solid #e5e5e5;
  padding-top: 1rem;
}

:deep(.video-js .vjs-big-play-button) {
  left: 50% !important;
  top: 50% !important;
  transform: translate(-50%, -50%) !important;
}

.modal-card {
  width: 85%;
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.missing {
  opacity: 0.6;
}

.block-tab-content {
  flex: 1 1 auto;
}

.block-info {
  background: var(--bg-card);
  border-radius: var(--radius-md);
  padding: 1em;
  border: 1px solid var(--border-color);
}

.block-tags {
  max-height: 200px;
  overflow: scroll;
  scrollbar-width: none;
}

.scene-details-page .block-tags {
  max-height: none;
  overflow: visible;
}

.block-tags::-webkit-scrollbar {
  display: none;
}

.block-opts {
}

.vue-star-rating {
    line-height: 0;
}

.scene-id {
  position: absolute;
  right: 10px;
  bottom: 5px;
  font-size: 11px;
  color: var(--text-muted);
}

.scene-details-page .scene-id {
  position: static;
  margin-top: 1rem;
  text-align: right;
}

.prev, .next {
  cursor: pointer;
  position: absolute;
  top: 50%;
  width: auto;
  padding: 16px;
  margin-top: -50px;
  color: white;
  font-weight: bold;
  font-size: 24px;
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  user-select: none;
  -webkit-user-select: none;
  background: rgba(0, 0, 0, 0.3);
  transition: background var(--transition-fast);
}
.prev:hover, .next:hover {
  background: rgba(108, 92, 231, 0.5);
}

.next {
  right: 0;
  border-radius: var(--radius-sm) 0 0 var(--radius-sm);
}

.prev {
  left: 0;
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
}

span.is-active img {
  border: 2px;
}

.pathDetails {
  color: var(--text-muted);
}

.heatmapFunscript {
  width: 100%;
  padding: 0;
  margin-top: 0.5em;
}

.heatmapFunscript img {
  border: 1px solid var(--border-color);
  width: 100%;
  height: 20px;
  margin: 0;
  padding: 0;
  border-radius: 4px;
}
.videosize {
  color: var(--text-secondary);
  font-weight: 550;
}

:deep(.carousel .carousel-indicator) {
  justify-content: flex-start;
  width: 100%;
  max-width: min-content;
  margin-left: auto;
  margin-right: auto;
  overflow: auto;
}
:deep(.carousel .carousel-indicator .indicator-item:not(.is-active)) {
  opacity: 0.5;
}
.is-divider {
  margin: .8rem 0;
}
.image-row {
  display: flex;
}
.image-wrapper {
  position: relative;
}
.thumbnail {
  height: 100px;
  margin-right: .5em;
  object-fit: cover;
  border-radius: var(--radius-sm);
  transition: transform var(--transition-fast);
}
.thumbnail:hover {
  transform: scale(1.05);
}
.tooltip {
  position: absolute;
  z-index: 1;
  top: 50px;
  right: 100%;
  width: 400px;
  height: 400px;
  background-color: var(--bg-card);
  box-shadow: var(--shadow-lg);
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 10px;
  transform: translateX(10px);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
}
.tooltip img {
  max-width: 100%;
  max-height: 100%;
}
.altsrc-image-wrapper {
  display: inline-block;
  margin-left: 5px;
}</style>
