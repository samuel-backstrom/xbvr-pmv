<template>
  <div class="container">
    <div class="content">
      <h3>{{ $t('Logging') }}</h3>
      <hr />

      <p>
        XBVR writes logs to <code>xbvr.log</code> in the application data directory.
      </p>
      <p>
        For more verbose logs, restart XBVR with debug environment flags:
      </p>
      <pre><code>DEBUG=true
DEBUG_REQUESTS=true
DEBUG_SQL=true
DEBUG_WS=true</code></pre>

      <p>
        Tip: use <code>tail -f xbvr.log</code> in the app data directory to watch logs live.
      </p>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Logging'
}
</script>
