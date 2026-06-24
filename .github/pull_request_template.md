## Summary

<!-- What changed and why? One or two sentences on the behaviour, not a file list. -->

## Related issue

<!-- Link: Fixes #123, or "N/A" for drive-by fixes -->

## Type of change

- [ ] Bug fix (non-breaking)
- [ ] New feature / enhancement
- [ ] Documentation or examples only
- [ ] Refactor / chore (no behaviour change)
- [ ] Breaking change (describe migration below)

## Behaviour

<!-- What does the user / operator see differently? Example CLI output, exit codes, config schema. -->

## Test plan

<!-- Check what you ran. Paste output for non-trivial changes. -->

- [ ] `make fmt vet test`
- [ ] Tested offline (`--mock`), e.g. `make run-plan`
- [ ] Tested against live RPC (if RPC/config/chain path changed)
- [ ] Updated docs / examples (if user-facing)

<details>
<summary>Test output (optional)</summary>

```text
Paste relevant command output here
```

</details>

## Checklist

- [ ] I have read the [Contributing guide](../CONTRIBUTING.md) and [Code of Conduct](../CODE_OF_CONDUCT.md)
- [ ] Planning path still does not send transactions (if touching `plan` / resources / export)
- [ ] New behaviour includes tests where it makes sense
- [ ] No secrets, RPC URLs, or API keys in committed configs
