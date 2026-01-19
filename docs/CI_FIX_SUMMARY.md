# CI Detection Priority Fix - Summary

## Issue
CI workflow had inconsistent app type detection priority compared to scripts:
- **CI (old)**: `.apptype` → `metadata.yaml` → file detection
- **Scripts**: `metadata.yaml` → `.apptype` → file detection

## Solution
Updated `.github/workflows/ci.yml` to match scripts priority:
1. `metadata.yaml` (preferred)
2. `.apptype` (legacy support)
3. File characteristics (fallback)

## Changes Made
Updated 3 jobs in CI workflow:
- `build-apps` job (line ~140)
- `push-images` job (line ~305)
- `security-scan` job (line ~569)

## Impact
- CI now prioritizes `metadata.yaml` over `.apptype` files
- Consistent behavior between CI and local scripts
- Backward compatible with existing `.apptype` files
- No breaking changes to existing workflows

## Verification
All detection blocks now show:
```bash
# Priority 1: Check metadata.yaml (preferred)
# Priority 2: Check .apptype file (legacy support)
# Priority 3: Detect by file characteristics
```

## Related Documentation
- `docs/METADATA_MIGRATION.md` - Migration guide
- `docs/SHORT_NAMES_REFERENCE.md` - Short name conventions
- `scripts/app-manager.sh` - Reference implementation
