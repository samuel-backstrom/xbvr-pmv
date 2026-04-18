# Files Filter and Query Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore the legacy files-section filtering and query behavior in the new UI so operators can narrow files by state, date, resolution, bitrate, framerate, and filename using the backend.

**Architecture:** Keep the work inside `ui-new/src/pages/Files.tsx` and the existing `getFiles()` API helper. The page should own a single query model, convert it into `/api/files/list` params, and reload when filters change. Local filtering should only remain where the backend has no equivalent, such as the existing type toggle.

**Tech Stack:** React, TypeScript, Framer Motion, date-fns, the existing `ui-new/src/api/client.ts` helper.

---

### Task 1: Define the file query state and fetch path

**Files:**
- Modify: `ui-new/src/pages/Files.tsx:1-240`

- [ ] **Step 1: Add a focused testable query model to the page**

```tsx
type FileState = 'all' | 'matched' | 'unmatched'
type SortField = 'filename' | 'created_time' | 'size' | 'video_width' | 'video_height' | 'video_bitrate' | 'duration' | 'video_avgfps_val'
type SortDir = 'asc' | 'desc'
```

- [ ] **Step 2: Fetch files from the backend when the query model changes**

```tsx
useEffect(() => {
  setLoading(true)
  getFiles({
    state: fileState === 'all' ? undefined : fileState,
    sort: `${sortField}_${sortDir}`,
    resolutions,
    framerates,
    bitrates,
    filename: debouncedFilename || undefined,
    createdDate: createdRange.start && createdRange.end ? [createdRange.start, createdRange.end] : undefined,
  })
    .then((data) => setFiles(data || []))
    .catch(() => setFiles([]))
    .finally(() => setLoading(false))
}, [fileState, sortField, sortDir, resolutions, framerates, bitrates, debouncedFilename, createdRange])
```

- [ ] **Step 3: Verify the page still builds**

Run: `npm --prefix ui-new run build`
Expected: exit 0 with a successful TypeScript and Vite build.

### Task 2: Restore the legacy filter controls

**Files:**
- Modify: `ui-new/src/pages/Files.tsx:1-240`

- [ ] **Step 1: Add the missing filter controls above the table**

```tsx
<button onClick={() => setFileState('matched')}>Matched</button>
<button onClick={() => setFileState('unmatched')}>Unmatched</button>
<button onClick={() => setCreatedRange({ start: formatDateInput(subDays(new Date(), 7)), end: formatDateInput(new Date()) })}>Last 7 days</button>
```

- [ ] **Step 2: Keep the existing type toggle as a local-only convenience filter**

```tsx
const visibleFiles = files.filter((file) => typeFilter === 'all' || file.type === typeFilter)
```

- [ ] **Step 3: Verify the controls update the visible file set without breaking the page**

Run: `npm --prefix ui-new run build`
Expected: exit 0.

### Task 3: Add the missing date helpers and sort helpers

**Files:**
- Modify: `ui-new/src/pages/Files.tsx:1-240`

- [ ] **Step 1: Add local date formatting helpers for `<input type="date">`**

```tsx
function formatDateInput(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
```

- [ ] **Step 2: Add sort labels that match backend sort fields**

```tsx
const SORT_OPTIONS = [
  { label: 'Filename', field: 'filename' },
  { label: 'Created time', field: 'created_time' },
  { label: 'Size', field: 'size' },
  { label: 'Width', field: 'video_width' },
  { label: 'Height', field: 'video_height' },
  { label: 'Bitrate', field: 'video_bitrate' },
  { label: 'Duration', field: 'duration' },
  { label: 'FPS', field: 'video_avgfps_val' },
]
```

- [ ] **Step 3: Verify the sort change updates request parameters**

Run: `npm --prefix ui-new run build`
Expected: exit 0.

### Task 4: Verify the new files page in the browser

**Files:**
- Modify: none

- [ ] **Step 1: Load `http://localhost:9999/ui/files` and confirm the filter UI renders**

- [ ] **Step 2: Confirm the page can request matched and unmatched files**

- [ ] **Step 3: Confirm created-date presets change the backend results**

