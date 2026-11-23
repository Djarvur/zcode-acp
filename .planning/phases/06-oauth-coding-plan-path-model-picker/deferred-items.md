# Deferred Items — Phase 6

## Deferred Items

- 06-05: pre-existing `mise run lint` failures the wave-1 merge shipped, out of
  06-05's scope (files/functions this plan did not author):
  status: acknowledged
  - front.go NewSession funlen (70 > 60 — 06-04's widened function)
  - snapshot.go readBuiltinProviderRules funlen (61 > 60) + err113 dynamic
    error + tagliatelle camelCase wire keys ×4 (06-03's capture-verbatim dig)
  - acpfront_test package goconst: "zai-api/GLM-5.3" ×7 (05-07's
    authFixWantModels + 06-04's newsession/configoptions fixtures) and
    "MethodNotFound" ×3 (05-07's authScenario + 06-04's newsession fixtures)
- 06-05: familysel.go tagliatelle ×2 — capture-verbatim camelCase wire keys
  (providerFamilyConnectionSelections / modelProviderFamilyModes): the
  desktop-read discipline REQUIRES these exact keys; same accepted class as
  snapshot.go's pre-existing hits.
  status: acknowledged
