# Country catalog and region-adaptive scoring

New configurations default to automatic scoring; `scoreFactor` alone does not
select the manual curve. Manual selection requires explicit `scoringMode`, not
a UI-specific `advanced` flag. Sampling probabilities and bonuses are separate
from the regional decay scale.

## Public catalog

Public `GET /countryCatalog` (and CORS `OPTIONS`) returns:

```ts
type CountryCatalog = {
  countries: {
    code: string
    name: string
    continent: 'AF' | 'AS' | 'EU' | 'NA' | 'OC' | 'SA'
    center: { lat: number; lng: number }
    spanKm: number
  }[]
  continents: { id: string; name: string; codes: string[] }[]
  scoring: {
    worldSpanKm: number
    worldHalfKm: number
    minimumHalfKm: 1
    perfectMeters: 100
  }
}
```

`GET /countryList` remains the existing name-sorted `string[]`, without a wrapper
or changed names. Catalog countries use that same order; continent IDs and their
codes are sorted. Values are finite, centers have both axes, and `spanKm` is
positive and at most `worldSpanKm`. Catalog access returns detached copies.

`assets/continents.json` is the reusable authoritative application placement
file, derived from [UN M49](https://unstats.un.org/unsd/methodology/m49/overview/),
with explicit geographic choices:

- Russia (`RU`) is Europe; Turkey (`TR`) and Kazakhstan (`KZ`) are Asia. A
  transcontinental country is selected whole, not split into different playable
  polygons per continent.
- North America includes Central America and the Caribbean. Greenland, Bermuda,
  Martinique, Saint Pierre and Miquelon, Puerto Rico and the US Virgin Islands
  are North America, independent of their sovereign state's continent.
- Réunion and British Indian Ocean Territory follow M49 Africa. Falkland Islands
  and South Georgia/South Sandwich Islands follow M49 South America.
- Cocos and Christmas Islands are Asia (geography near Southeast Asia), an
  explicit exception to M49's Australia/New Zealand grouping. Taiwan is Asia;
  the M49 table has no separate Taiwan row. The supported US Minor Outlying
  Islands asset is Midway Atoll, in Oceania.
- Only supported country codes appear. No Antarctica option is manufactured.
  Startup rejects unknown, duplicated, missing countries or missing/invalid
  continent definitions, rather than returning a partial catalog. Current
  counts: AF 17, AS 28, EU 46, NA 13, OC 7, SA 10; total 121.

The names are the existing country metadata names, not a new translated naming
system. The local visual selector may localize display names without changing
codes or continent membership.

## Geographical scale, independent of sampling

For each country, use all existing `assets/basic/{CC}.json` playable rings:

1. Validate coordinate pairs, finite geographic ranges, nonempty polygons,
   positive sampling size/search radius, at least three ring points and nonzero
   unwrapped ring area. This is bounded asset validation, not a general topology
   repair/self-intersection engine.
2. Sort longitudes around the globe and remove their largest circular gap.
   The remaining envelope handles the antimeridian; center is its longitude
   midpoint and the midpoint of the latitude extrema.
3. Find the maximum spherical distance from that center to the envelope,
   checking corners and interior latitude extrema of the longitude edges.
   Checking only corners can underbound wide high-latitude regions. Include
   the largest polygon Street View search radius (meters converted to km).
4. `spanKm = min(pi * 6371, 2 * (envelopeRadiusKm + searchRadiusKm))`.

This conservative geographical extent is not administrative area, a capital or
an exact polygon diameter. It includes islands/remote playable areas and may
have a center outside playable land. Derivation is O(V log V), not all vertex
pairs. Country metadata `Size` stays raw; its existing `pow(Size, 0.16)` sampling
weight is now a separate field. Polygon sampling still uses `sqrt(size)`.
Region/configuration changes never rewrite those weights.

For normalized selected codes:

```text
worldSpanKm = pi * 6371
selectionSpanKm = min(worldSpanKm,
  max(each country.spanKm,
      each pair's haversine(centerA, centerB) + (spanA + spanB) / 2))
```

Haversine distance uses radius **6371 km**, radians and a `[0,1]` clamp on its
intermediate value. For `[]` or every supported code, use `worldSpanKm` exactly.
Pair work is bounded to 256 unique countries. Geographically separated custom
countries therefore do not inherit an artificially tiny summed-area curve.

## Configuration, updates and lifecycle

`LobbyConf` adds these fields to every public/member projection:

```ts
scoringMode: 'auto' | 'manual'
scoreScaleKm: number // READ ONLY: effective half-score decay distance in km
scoreFactor?: number // unchanged manual input, integer 1..500
```

- Creation without `scoringMode` uses `auto`, even if `scoreFactor` is present.
- `manual` is an explicit advanced override. Omitted/zero `scoreFactor` keeps
  its existing default/no-update interpretation; new distance-mode defaults
  remain 100. Factors outside 1..500 (except the existing omitted/zero sentinel)
  and unknown policies reject before mutation.
- Submitted finite `scoreScaleKm` is ignored and recomputed, regardless of its
  magnitude/sign. Nonfinite numbers are rejected (JSON rejects these as well).
  It is not an extra client-controlled tuning parameter.
- Missing/empty `scoringMode` on an **update** preserves the current policy,
  unless the effective country selection actually changes, then it resets to
  `auto`. Explicit `manual` wins even when the region also changes. A supplied
  score factor alone does not change policy.
- Codes must match supported uppercase identifiers exactly. Duplicates are
  normalized to sorted unique codes, not extra sampling weight. More than
  256 raw entries rejects with `INVALID_LOBBY_SETTINGS`; unknown codes return
  `INVALID_COUNTRY_CODE`. Existing 64KiB HTTP and 16KiB socket limits remain.
- Omitted/null update `ccList` means no country change; explicit `[]` is the
  existing **world** contract. Reordering/deduplication is not a region change.
  Explicit all-catalog codes and `[]` are equivalent world selections. There
  is no `regionMode` field: the frontend must reject an empty Custom choice
  before submission, not silently fall back to world.
- Merge into a detached candidate under the existing lobby mutex, normalize
  and derive the complete candidate, then commit configuration/sampling sum
  together. Invalid requests never partially change settings. Ready flags,
  locks, admission gates and player/result state are untouched by settings.
- Creation and updates normalize authoritative configuration. Round preparation
  (including replay) derives a detached normalized configuration again; it is
  committed only after successful generation/ownership/provider validation.
  Failed replay selection keeps the completed summary/configuration intact.
  Recovery/public snapshots clone the stored normalized values without trusting
  new client input or recalculating historical scores.
- Country Battle (`mode:2`) still uses its original time/country formula and
  has `scoreScaleKm:0`, no score factor. The policy string is retained but has no
  effect there. Switching back to distance mode derives a fresh effective scale
  (default factor 100 unless supplied), respecting the selected policy.

Example requests (not live socket captures):

```json
{"command":"update_lobby_settings","conf":{"ccList":["SI"],"scoringMode":"auto"}}
{"command":"update_lobby_settings","conf":{"ccList":["SI","NZ"],"scoringMode":"auto"}}
{"command":"update_lobby_settings","conf":{"scoringMode":"manual","scoreFactor":1}}
{"command":"update_lobby_settings","conf":{"scoringMode":"auto"}}
```

## Curves and exact frontend preview

```text
worldHalfKm = ln(0.5) / ln(0.999)
              / (0.2 + 30 * pow(0.98, 10 * pow(100, 0.6)))
halfKm = max(1, worldHalfKm * min(worldSpanKm, selectionSpanKm) / worldSpanKm)
basePoints = min(5000,
  floor(5000 * pow(0.5, max(0, distanceMeters - 100) / (halfKm * 1000))))
```

World/all-catalog uses the exact `worldHalfKm`, avoiding round-trip arithmetic
at that boundary. The catalog publishes all primitives necessary for a local
preview; use the returned unrounded numbers. `scoreScaleKm` on the accepted
lobby is authoritative. Half-score distance from the answer is
`0.1 + scoreScaleKm` km because automatic scoring retains the 100m perfect
radius. The 1km decay floor is intentional for microregions. Base points exclude
placement, double and duel bonuses.

Manual mode retains the **unchanged** `scoreDistance(distanceMeters, factor)`:

```text
min(5000, floor(5000 * pow(0.999,
  (distanceMeters/1000 - factor/1000)
    * (0.2 + 30 * pow(0.98, 10 * pow(factor, 0.6))))))
```

Its perfect radius is `factor` meters (legacy behavior), not the automatic
100m radius. Its read-only decay distance is
`ln(0.5) / ln(0.999) / (0.2 + 30 * pow(0.98, 10 * pow(factor,0.6)))` km.
The old factor-1 minimum has half-distance **28.034776019221 km**; automatic
Slovenia can be substantially stricter without changing that legacy formula.
