## Summary

<!-- What does this PR do? 1-3 bullet points. -->

-
-

## Type of change

- [ ] `fix:` Bug fix
- [ ] `feat:` New feature
- [ ] `enhance:` Enhancement / improvement
- [ ] `refactor:` Code restructuring (no behaviour change)
- [ ] `test:` Tests only
- [ ] `docs:` Documentation only
- [ ] `chore:` Maintenance / dependencies

## Checklist

- [ ] `make lint` passes (govulncheck + gosec + golangci-lint)
- [ ] `make build` succeeds
- [ ] `make test` passes
- [ ] New code has unit tests in the appropriate `*_test.go` file
- [ ] All inputs sanitized via `utils.Sanitize()` / `utils.SanitizePath()`
- [ ] No `exec.Command` with unsanitized user input
- [ ] No sensitive data or secrets committed

## Testing notes

<!-- How was this tested? Mention any edge cases covered or skipped. -->

## Related issues

<!-- Closes #, Fixes #, or N/A -->
