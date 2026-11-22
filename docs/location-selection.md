# Location selection and country coverage

## Three kinds of geography

- **Availability and grouping:** keys in `assets/countryDB.json` define supported
  countries and territories; `assets/continents.json` places each exactly once.
  `/countryList` and `/countryCatalog` publish this same supported set. World means
  all supported countries, not every country where Google has imagery.
- **Sampling areas:** `assets/basic/{CC}.json` contains named search polygons,
  their raw `size`, a metadata search `radius` in meters and coordinate rings in
  `[longitude, latitude]` order. The first ring is the outer boundary; later
  rings exclude holes. These areas are deliberately smaller than some national
  outlines and are also the basis of automatic scoring extents.
- **National outlines:** `assets/full/countries2.json`, grouped by `ISO_A2_EH`,
  supplies country reveals and the frontend's generated selection map. Repeated
  country codes retain every map unit. `assets/reverse.json` independently
  classifies Country Battle guesses. Decorative outlines are not coverage maps.

The frontend reads availability from the backend. Its region map and initial
single-country camera are bundled assets, regenerated when supported codes
change; see the matching frontend release's region-settings guide. Normal
frontend builds require no sibling backend checkout.

## How a round location is selected

1. Choose a supported country using `pow(country.size, 0.16)`. An explicit
   country selection restricts this draw; the empty list means World.
2. Choose one of that country's named areas using `int(sqrt(area.size))`.
   These weights are computed separately from the catalog's geographical scale.
3. Uniformly sample longitude/latitude inside the area's bounding rectangle,
   rejecting points outside the outer ring or inside a hole. This is **land-area
   sampling, not road sampling**, and is not equal-area spherical sampling.
4. Ask Google's Street View **metadata** endpoint for nearby outdoor imagery,
   using the area's radius. A successful response supplies the round's answer
   coordinates. `source=outdoor` is not a promise of a public road, a particular
   imagery contributor, or an interesting scene.
5. For countries with `strictBounds: true` in their country metadata, accept the
   returned point only if it remains inside the **chosen** area, outside its
   holes, and inside one of that country's full-outline polygons, respecting
   their holes too. An outside point is a coverage miss, not a valid answer or a
   reason to relabel the country. Full bounds are decoded once at startup.

Legacy countries omit `strictBounds` and retain their existing behavior: their
areas constrain the candidate, but the metadata radius can return a point outside
those search areas. Enabling strict bounds for an existing country is a deliberate
coverage change and should be assessed separately; do not enable it globally as
an incidental data cleanup.

Selection has one shared limit of **8 metadata requests, 1,024 candidate points
and 6 seconds**, with a **2-second** per-request timeout. The same polygon is tried
for up to four metadata requests before another country/area draw, which can
choose the same area again. Outside strict bounds consumes the existing attempt,
not an extra budget. Missing coverage and transient provider failures can retry;
authentication/quota failures stop immediately. Exhaustion fails safely before
committing a round; no unchecked coordinate or arbitrary country is substituted.

Google documents [metadata requests](https://developers.google.com/maps/documentation/streetview/metadata)
as **free and not consuming Static API quota**. Misses still add network latency.
They are not static image downloads. Browser map/panorama loading uses separate
[Maps JavaScript API SKUs](https://developers.google.com/maps/documentation/javascript/usage-and-billing);
retain credential, spending and live-game verification gates.

## Kazakhstan's initial coverage policy

`KZ` is grouped under Asia, including its transcontinental national outline. Its
initial search areas are compact urban windows in **Aktobe, Almaty, Astana, Semey
and Shymkent**, rather than the country's large areas of uninhabited steppe.

[Google's Kazakhstan launch announcement](https://blog.google/products-and-platforms/products/maps/google-street-view-kazakhstan/)
explicitly names these five cities and describes more than 42,000 km captured.
That establishes city-level availability, **not coverage of every street**.
The windows are manually curated around city centers located with
[OpenStreetMap/Nominatim](https://nominatim.openstreetmap.org/), using public
motor-vehicle road context from [OpenStreetMap](https://www.openstreetmap.org/copyright)
to avoid sparse outer administrative boundaries. OpenStreetMap data is available
under the ODbL. Neither service is called at runtime, and road presence is not a
Street View availability or image-quality measurement.

| Area     | West / south      | East / north      | Raw size |
| -------- | ----------------- | ----------------- | -------- |
| Aktobe   | 57.1973 / 50.2561 | 57.2533 / 50.2921 | 16       |
| Almaty   | 76.9177 / 43.2184 | 76.9737 / 43.2544 | 18       |
| Astana   | 71.4025 / 51.1103 | 71.4585 / 51.1463 | 16       |
| Semey    | 80.2223 / 50.4087 | 80.2783 / 50.4447 | 16       |
| Shymkent | 69.5603 / 42.2967 | 69.6163 / 42.3327 | 18       |

Each uses a **750m** metadata search radius and strict returned-point bounds.
Raw sizes approximate window area in square kilometers, rounded to integers;
country size **84** is their sum, not Kazakhstan's administrative area. Existing
integer-square-root weighting gives each window weight 4: equal **20% initial
area draws**. Successful rounds can still be biased by differing imagery hit
rates. Automatic scoring uses the extent and separation of all five windows,
not their summed size or the decorative national outline.

This is an intentionally urban first set, not nationwide road coverage or a
promise that every returned panorama is an interesting street. City parks and
other outdoor scenes can still exist within the windows. Add reviewed towns or
narrow connecting-road corridors for variety rather than opening the whole
national polygon or raising the search radius indiscriminately.

## Adding or refreshing a country

1. Confirm published outdoor coverage, preferring Google's actual launch or
   expansion announcement over a future collection schedule.
2. Add or update a small set of named, non-overlapping dense urban/town areas or
   narrow road corridors. Use holes for known large water/empty exclusions when
   needed. Keep sizes positive integers and radii conservative. The current
   weights are square roots: splitting one area into many changes its total
   probability, and overlap double-counts places. They do not express an explicit
   urban/rural percentage; assess a new weighting policy separately if needed.
3. For a new country, set `strictBounds: true` and ensure areas lie inside its
   local full-country outlines. Verify that the reverse dataset also contains
   the code. Full national boundaries are not a substitute for sampled coverage.
4. Add metadata and exactly one continent membership. Catalog/raw-input capacity
   is a fixed **256**, independent of the current supported count; unknown codes
   remain invalid. Keep the HTTP/socket byte limits and bounded selection intact.
5. Regenerate frontend region-map and country-viewports from the matching backend
   data. Run both generators' `--check` commands and each repository's pinned
   static/type/build checks. Check the outline and initial country framing.
6. Assess real metadata hit rates and panorama variety only with explicitly
   approved provider scope. Offline geometry/road checks cannot certify imagery
   availability. Do not store a permanent panorama-ID database or add a live road
   service merely to expand the supported set.

Distance-mode guesses remain valid anywhere with finite in-range coordinates;
search masks restrict **answer generation**, not where a player may place a pin.
Scoring uses distance to the actual returned answer and the server-derived region
curve. Country Battle uses the separate local reverse lookup and remains an
unfinished game mode; this coverage policy does not complete it.
