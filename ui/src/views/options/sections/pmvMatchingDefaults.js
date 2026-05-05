export function selectPMVImportVolume (volumes, currentPathPrefix = '', currentVolumeId = 0) {
  const localVolumes = Array.isArray(volumes)
    ? volumes.filter(volume => volume && volume.path && (!volume.type || volume.type === 'local'))
    : []

  if (localVolumes.length === 0) {
    return {
      pathPrefix: currentPathPrefix || '',
      volumeId: currentVolumeId || 0
    }
  }

  const currentPath = typeof currentPathPrefix === 'string' ? currentPathPrefix.trim() : ''
  const currentVolume = localVolumes.find(volume => (
    (currentVolumeId > 0 && volume.id === currentVolumeId) ||
    (currentPath && volume.path === currentPath)
  ))

  if (currentVolume) {
    return {
      pathPrefix: currentVolume.path,
      volumeId: currentVolume.id || 0
    }
  }

  return {
    pathPrefix: localVolumes[0].path,
    volumeId: localVolumes[0].id || 0
  }
}
