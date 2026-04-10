# XBVR PMV - WSL Runbook

## Permanent WSL data location
- App/data dir: `/home/samuel/.local/share/xbvr-pmv-wsl`
- Database file: `/home/samuel/.local/share/xbvr-pmv-wsl/main.db`
- Video volume path (WSL): `/mnt/g/Videos`

## Start server (WSL, persistent DB)
Preferred: run the persistent binary.

```bash
/home/samuel/.local/share/xbvr-pmv-wsl/xbvr-server -app_dir /home/samuel/.local/share/xbvr-pmv-wsl
```

Alternative (dev only, slower startup due to build each run):
```bash
go run . -app_dir /home/samuel/.local/share/xbvr-pmv-wsl
```

## Build/update persistent binary
Run from repo root:

```bash
go build -o /home/samuel/.local/share/xbvr-pmv-wsl/xbvr-server .
```

## Verify server + storage
```bash
curl -sS -i http://127.0.0.1:9999/ | head -n 20
curl -sS http://127.0.0.1:9999/api/options/storage
```

Expected storage volume:
- `path: /mnt/g/Videos`
- `is_available: true`

## First-time setup for a fresh DB
If volume is not yet configured:

```bash
curl -sS -i -H 'Content-Type: application/json' \
  -X POST http://127.0.0.1:9999/api/options/storage \
  -d '{"type":"local","path":"/mnt/g/Videos"}'
```

Then trigger scan:

```bash
curl -sS -i http://127.0.0.1:9999/api/task/rescan
```

## Notes
- Do not use `-localstorage` for this workflow; it creates/uses a temporary runtime DB.
- Windows-style path `G:\Videos` is not valid for WSL runtime. Use `/mnt/g/Videos`.
- Persistent binary path: `/home/samuel/.local/share/xbvr-pmv-wsl/xbvr-server`
