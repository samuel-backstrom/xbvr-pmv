<template>
  <div>
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keypress.prevent.questionMark="$store.commit('overlay/showQuickFind')"
    />
    <Navbar/>
    <div class="navbar-pad">
      <router-view/>
    </div>

    <Details v-if="showOverlay"/>
    <EditScene v-if="showEdit" />
    <ActorDetails v-if="showActorDetails"/>
    <EditActor v-if="showActorEdit" />
    <SearchStashdbScenes v-if="showSearchStashdbScenes" />
    <SearchStashdbActors v-if="showSearchStashdbActors" />

    <QuickFind/>
    <MigrationOverlay/>

    <Socket/>
  </div>
</template>

<script>
import GlobalEvents from 'vue-global-events'

import Navbar from './Navbar.vue'
import Socket from './Socket.vue'
import QuickFind from './QuickFind'
import Details from './views/scenes/Details'
import EditScene from './views/scenes/EditScene'
import ActorDetails from './views/actors/ActorDetails'
import EditActor from './views/actors/EditActor'
import SearchStashdbScenes from './views/scenes/SearchStashdbScenes'
import SearchStashdbActors from './views/actors/SearchStashdbActors'
import MigrationOverlay from './components/MigrationOverlay'

export default {
  components: { Navbar, Socket, QuickFind, GlobalEvents, Details, EditScene, ActorDetails, EditActor, SearchStashdbScenes, SearchStashdbActors, MigrationOverlay },
  computed: {
    showOverlay () {
      return this.$store.state.overlay.details.show
    },
    showEdit () {
      return this.$store.state.overlay.edit.show
    },
    showActorDetails() {
      return this.$store.state.overlay.actordetails.show
    },
    showActorEdit() {
      return this.$store.state.overlay.actoredit.show
    },
    showSearchStashdbScenes() {
      return this.$store.state.overlay.searchStashDbScenes.show
    },
    showSearchStashdbActors() {
      return this.$store.state.overlay.searchStashDbActors.show
    },
  }
}
</script>

<style>
  /* ===== GLOBAL DARK THEME ===== */
  :root {
    --bg-primary: #0f1117;
    --bg-secondary: #1a1d27;
    --bg-card: #1e2130;
    --bg-card-hover: #252838;
    --bg-surface: #252a3a;
    --bg-input: #1a1d27;
    --border-color: #2d3148;
    --border-light: #383d56;
    --text-primary: #e8eaf0;
    --text-secondary: #9ca3b8;
    --text-muted: #6b7394;
    --accent-primary: #6c5ce7;
    --accent-primary-hover: #7d6ff0;
    --accent-info: #4facfe;
    --accent-success: #00d68f;
    --accent-warning: #f0a500;
    --accent-danger: #ff6b6b;
    --shadow-sm: 0 2px 8px rgba(0, 0, 0, 0.3);
    --shadow-md: 0 4px 16px rgba(0, 0, 0, 0.4);
    --shadow-lg: 0 8px 32px rgba(0, 0, 0, 0.5);
    --radius-sm: 6px;
    --radius-md: 10px;
    --radius-lg: 16px;
    --transition-fast: 0.15s ease;
    --transition-base: 0.25s ease;
  }

  *, *::before, *::after {
    scrollbar-color: #383d56 transparent;
  }

  ::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }
  ::-webkit-scrollbar-track {
    background: transparent;
  }
  ::-webkit-scrollbar-thumb {
    background: #383d56;
    border-radius: 4px;
  }
  ::-webkit-scrollbar-thumb:hover {
    background: #4a5073;
  }

  html, body {
    background-color: var(--bg-primary) !important;
    color: var(--text-primary) !important;
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif !important;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
  }

  /* ===== BULMA / BUEFY OVERRIDES ===== */
  .label, label, strong, b, h1, h2, h3, h4, h5, h6 {
    color: var(--text-primary) !important;
  }
  .title, .subtitle {
    color: var(--text-primary) !important;
  }
  p, span, small, div {
    color: inherit;
  }
  a {
    color: var(--accent-info);
    transition: color var(--transition-fast);
  }
  a:hover {
    color: var(--accent-primary-hover);
  }

  .box, .card {
    background-color: var(--bg-card) !important;
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md) !important;
    box-shadow: var(--shadow-sm) !important;
    color: var(--text-primary);
  }

  /* Buttons */
  .button {
    border-radius: var(--radius-sm) !important;
    transition: all var(--transition-fast) !important;
    font-weight: 500;
  }
  .button.is-primary {
    background-color: var(--accent-primary) !important;
    border-color: var(--accent-primary) !important;
  }
  .button.is-primary:hover {
    background-color: var(--accent-primary-hover) !important;
    border-color: var(--accent-primary-hover) !important;
    box-shadow: 0 4px 12px rgba(108, 92, 231, 0.4) !important;
  }
  .button.is-info {
    background-color: var(--accent-info) !important;
    border-color: var(--accent-info) !important;
  }
  .button.is-light, .button.is-outlined {
    background-color: var(--bg-surface) !important;
    border-color: var(--border-color) !important;
    color: var(--text-primary) !important;
  }
  .button.is-light:hover, .button.is-outlined:hover {
    background-color: var(--bg-card-hover) !important;
    border-color: var(--border-light) !important;
  }

  /* Tags */
  .tag {
    border-radius: var(--radius-sm) !important;
    font-weight: 500;
    font-size: 0.7rem !important;
  }
  .tag.is-warning {
    background-color: rgba(240, 165, 0, 0.15) !important;
    color: var(--accent-warning) !important;
  }
  .tag.is-info {
    background-color: rgba(79, 172, 254, 0.15) !important;
    color: var(--accent-info) !important;
  }
  .tag.is-primary {
    background-color: rgba(108, 92, 231, 0.15) !important;
    color: var(--accent-primary-hover) !important;
  }
  .tag.is-danger {
    background-color: rgba(255, 107, 107, 0.15) !important;
    color: var(--accent-danger) !important;
  }
  .tag.is-success {
    background-color: rgba(0, 214, 143, 0.15) !important;
    color: var(--accent-success) !important;
  }

  /* Form controls */
  .input, .textarea, .select select, .taginput .taginput-container {
    background-color: var(--bg-input) !important;
    border-color: var(--border-color) !important;
    color: var(--text-primary) !important;
    border-radius: var(--radius-sm) !important;
    transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
  }
  .input:focus, .textarea:focus, .select select:focus,
  .input:active, .textarea:active, .select select:active {
    border-color: var(--accent-primary) !important;
    box-shadow: 0 0 0 3px rgba(108, 92, 231, 0.2) !important;
  }
  .input::placeholder, .textarea::placeholder {
    color: var(--text-muted) !important;
  }

  /* Checkbox & Radio */
  .b-checkbox.checkbox input[type=checkbox] + .check {
    border-color: var(--border-light) !important;
    background: var(--bg-input) !important;
  }
  .b-radio.radio input[type=radio] + .check {
    border-color: var(--border-light) !important;
  }
  .radio-button .button {
    background-color: var(--bg-surface) !important;
    border-color: var(--border-color) !important;
    color: var(--text-secondary) !important;
  }
  .radio-button input[type=radio]:checked + .button {
    background-color: var(--accent-primary) !important;
    border-color: var(--accent-primary) !important;
    color: #fff !important;
  }
  .checkbox-button .button {
    background-color: var(--bg-surface) !important;
    border-color: var(--border-color) !important;
    color: var(--text-secondary) !important;
  }
  .checkbox-button input[type=checkbox]:checked + .button {
    background-color: var(--accent-primary) !important;
    border-color: var(--accent-primary) !important;
    color: #fff !important;
  }

  /* Tabs */
  .b-tabs .tab-content {
    background: transparent !important;
  }
  .tabs a {
    color: var(--text-secondary) !important;
    border-bottom-color: transparent !important;
  }
  .tabs a:hover {
    color: var(--text-primary) !important;
    border-bottom-color: var(--accent-primary) !important;
  }
  .tabs li.is-active a {
    color: var(--accent-primary) !important;
    border-bottom-color: var(--accent-primary) !important;
    font-weight: 600;
  }
  .tabs ul {
    border-bottom-color: var(--border-color) !important;
  }

  /* Modals */
  .modal-background {
    background-color: rgba(0, 0, 0, 0.70) !important;
    backdrop-filter: blur(4px);
  }
  .modal-card {
    border-radius: var(--radius-lg) !important;
    overflow: hidden;
  }
  .modal-card-body {
    background-color: var(--bg-secondary) !important;
    color: var(--text-primary) !important;
  }
  .modal-card-head, .modal-card-foot {
    background-color: var(--bg-card) !important;
    border-color: var(--border-color) !important;
  }
  .modal-card-title {
    color: var(--text-primary) !important;
  }

  /* Dropdown & Menu */
  .dropdown-content, .menu {
    background-color: var(--bg-card) !important;
    border-radius: var(--radius-md) !important;
  }
  .dropdown-item, .dropdown .dropdown-menu .has-link a {
    color: var(--text-primary) !important;
  }
  .dropdown-item:hover, .dropdown-item.is-active {
    background-color: var(--bg-surface) !important;
    color: var(--accent-info) !important;
  }
  .menu-list a {
    color: var(--text-secondary) !important;
    border-radius: var(--radius-sm) !important;
    transition: all var(--transition-fast);
  }
  .menu-list a:hover {
    background-color: var(--bg-surface) !important;
    color: var(--text-primary) !important;
  }
  .menu-list a.is-active {
    background-color: var(--accent-primary) !important;
    color: #fff !important;
  }
  .menu-label {
    color: var(--text-muted) !important;
    text-transform: uppercase;
    font-size: 0.7rem !important;
    letter-spacing: 0.08em;
  }

  /* Tables */
  .table {
    background-color: var(--bg-card) !important;
    color: var(--text-primary) !important;
  }
  .table td, .table th {
    border-color: var(--border-color) !important;
    color: var(--text-primary) !important;
  }
  .table tr:hover {
    background-color: var(--bg-surface) !important;
  }

  /* Pagination */
  .pagination-previous, .pagination-next, .pagination-link {
    background-color: var(--bg-surface) !important;
    border-color: var(--border-color) !important;
    color: var(--text-secondary) !important;
    border-radius: var(--radius-sm) !important;
  }
  .pagination-link.is-current {
    background-color: var(--accent-primary) !important;
    border-color: var(--accent-primary) !important;
    color: #fff !important;
  }

  /* Tooltips */
  .b-tooltip .tooltip-content {
    background-color: var(--bg-card) !important;
    color: var(--text-primary) !important;
    border-radius: var(--radius-sm) !important;
    box-shadow: var(--shadow-md) !important;
    font-size: 0.8rem;
  }

  /* Slider */
  .b-slider .b-slider-fill {
    background: var(--accent-primary) !important;
  }
  .b-slider .b-slider-track {
    background: var(--border-color) !important;
  }
  .b-slider .b-slider-thumb-wrapper .b-slider-thumb {
    background: var(--accent-primary) !important;
    border-color: var(--accent-primary) !important;
  }

  /* Dividers */
  .is-divider, .is-divider[data-content]::after {
    border-top-color: var(--border-color) !important;
    color: var(--text-muted) !important;
  }

  /* Loading */
  .loading-overlay .loading-background {
    background: rgba(15, 17, 23, 0.7) !important;
  }

  /* Notification / Snackbar */
  .notices .snackbar {
    background-color: var(--bg-card) !important;
    color: var(--text-primary) !important;
    border-radius: var(--radius-md) !important;
    box-shadow: var(--shadow-lg) !important;
  }

  /* Field label */
  .field .label, .field.has-addons .label {
    color: var(--text-secondary) !important;
    font-weight: 500;
    font-size: 0.85rem;
  }

  /* Autocomplete dropdown */
  .autocomplete .dropdown-menu .dropdown-content {
    background-color: var(--bg-card) !important;
    border: 1px solid var(--border-color) !important;
    border-radius: var(--radius-md) !important;
  }
  .autocomplete .dropdown-item {
    color: var(--text-primary) !important;
  }
  .autocomplete .dropdown-item:hover, .autocomplete .dropdown-item.is-hovered {
    background-color: var(--bg-surface) !important;
  }

  /* Carousel */
  .carousel .carousel-indicator .indicator-item .indicator-style {
    border-color: var(--border-color) !important;
  }

  /* ===== GLOBAL LAYOUT ===== */
  .navbar-pad {
    margin-top: 0.75em;
    padding: 0 0.5em;
  }

  /* Taginput */
  .taginput .taginput-container {
    background-color: var(--bg-input) !important;
    border-color: var(--border-color) !important;
  }
  .taginput .taginput-container.is-focusable:focus {
    border-color: var(--accent-primary) !important;
  }

  /* Toast */
  .toast {
    border-radius: var(--radius-md) !important;
  }

  /* Progress */
  .progress {
    background-color: var(--border-color) !important;
    border-radius: var(--radius-sm) !important;
  }

  /* Datepicker */
  .datepicker .dropdown-content {
    background-color: var(--bg-card) !important;
  }
  .datepicker .datepicker-header {
    background-color: var(--bg-card) !important;
    border-bottom-color: var(--border-color) !important;
  }
  .datepicker .datepicker-table .datepicker-body .datepicker-cell {
    color: var(--text-primary);
  }
  .datepicker .datepicker-table .datepicker-body .datepicker-cell.is-selectable:hover {
    background-color: var(--bg-surface);
    color: var(--text-primary);
  }
  .datepicker .datepicker-table .datepicker-body .datepicker-cell.is-selected {
    background-color: var(--accent-primary);
    color: #fff;
  }

  /* Field label-on-border */
  .field.has-addons .label,
  .field .label.is-floating-label {
    background: transparent;
  }

  /* Buefy switch */
  .switch input[type=checkbox] + .check {
    background: var(--border-color) !important;
  }
  .switch input[type=checkbox]:checked + .check {
    background: var(--accent-primary) !important;
  }

  /* Content area */
  .content h1, .content h2, .content h3, .content h4, .content h5, .content h6 {
    color: var(--text-primary) !important;
  }
  .content p {
    color: var(--text-secondary);
  }

  /* Card-image area */
  .card-image {
    overflow: hidden;
    position: relative;
  }

  /* Buefy field label on border dark fix */
  .field.is-floating-label .label,
  .field.has-addons > .label {
    background-color: var(--bg-primary);
    padding: 0 4px;
  }
</style>
