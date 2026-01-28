# Field Notes

Observed patterns from Entropia scans. Not claims about truth, just patterns in how information is presented and supported.

## Wikipedia Patterns

### Freshness Anomaly on Historical Topics

**Observation:** Historical topics (700+ years old) sometimes show median source age of <1 day.

**Example:** Borscht (soup documented since 1240s)
- 860 sources with age data
- Median age: 0.003 years (~1 day)
- All sources updated within last year

**Pattern:** When historical topics have all sources extremely recent, often indicates:
- Active content disputes
- Competing national origin claims
- Frequent back-and-forth edits
- Unstable narrative rather than established consensus

**Non-example:** Non-controversial historical topics typically have stable sources with varied ages.

---

### Navigation Link Contamination

**Observation:** Generic evidence extraction initially counted UI/navigation links as evidence.

**Before filtering:**
- Borscht article: 1177 evidence links
- ~200 were Wikipedia navigation (Main Page, Portal, Special:, Help:, Talk:, edit/history)
- Inflated evidence-to-claim ratio artificially

**After filtering:**
- Borscht article: 978 evidence links
- Only external sources and legitimate cross-references
- More accurate quality assessment

**Pattern:** Evidence extraction requires domain-specific filtering to avoid counting metadata as content.

---

### Historical Entity References in Modern Disputes

**Observation:** Articles about contested cultural/culinary origins often reference defunct states.

**Example:** Borscht article references:
- Kyivan Rus (ended 1240, 786 years ago)
- USSR (ended 1991, 35 years ago)
- Polish-Lithuanian Commonwealth (ended 1795, 231 years ago)
- Grand Duchy of Lithuania (ended 1795, 231 years ago)

**Pattern:** Using historical entities that no longer exist allows competing modern claims:
- "Originated in Kyivan Rus" (which modern nation inherits this?)
- "Soviet tradition" (which post-Soviet state owns this?)
- "Polish-Lithuanian" (Poland or Lithuania?)

**Hypothesis:** References to extinct states may correlate with modern identity disputes.

---

### Edit War Thresholds

**Implementation:** High conflict threshold:
- \>10 edits/month AND \>3 reverts, OR
- \>5 edits/day

**Observation:** Borscht article (clearly contested topic):
- 7 edits in last 30 days
- 0 reverts detected in last 100 revisions
- Below threshold

**Pattern:** Current threshold may be too conservative for detecting subtle ongoing disputes. Constant low-level editing may indicate persistent conflict without dramatic revert wars.

**Alternative signals identified:**
- Freshness anomaly (all sources very recent)
- Historical entity references
- Competing origin claims in text

---

## TLS Patterns

### HTTP Without Encryption

**Observation:** Pages served over HTTP (no TLS) trigger `no_tls` signal.

**Example:** http://example.com
- Signal: "Page served over HTTP without encryption" [WARNING]

**Pattern:** HTTP-only sites are increasingly rare for legitimate content sources. May indicate:
- Outdated/unmaintained sites
- Lower-quality sources
- Potential tampering vulnerability

---

### Certificate Expiration

**Observation:** Not yet observed in production scans (Wikipedia uses valid Let's Encrypt).

**Expected pattern:** Expired certificates likely indicate:
- Site no longer actively maintained
- Content possibly outdated
- Administrative neglect

---

## Source Quality Patterns

### High Citation Count ≠ Stable Narrative

**Traditional assumption:** More citations = better supported = more stable.

**Counter-observation:** Borscht article:
- 978 evidence links (high count)
- 35 claims (reasonable)
- Evidence-to-claim ratio: 27.94 (excellent)
- **BUT:** Median source age 1 day (unstable)

**Pattern:** Evidence quantity doesn't indicate narrative stability. Recent source updates on historical topics suggest ongoing disputes.

---

### Authority Distribution Patterns

**Wikipedia typical pattern:**
- High secondary source count (826 for Borscht)
- Low primary source count (6 for Borscht)
- Moderate tertiary count (146 for Borscht)

**Interpretation:** Wikipedia naturally acts as secondary source aggregator. Heavy reliance on secondary sources may:
- Amplify existing narratives
- Create distance from primary evidence
- Enable competing interpretations

---

## Methodology Notes

### What These Patterns Mean

These observations are **descriptive, not normative**. They describe:
- How information is structured
- How sources are cited
- How content changes over time

They do NOT determine:
- What is true
- What is correct
- What should be believed

### Pattern Confidence

**High confidence:**
- Navigation link contamination (directly measured)
- Freshness anomaly correlation with historical topics (repeatable)

**Medium confidence:**
- Historical entity correlation with disputes (limited sample)
- Edit war threshold effectiveness (needs tuning)

**Low confidence:**
- Long-term narrative stability (requires historical tracking)
- Cross-domain pattern generalization (needs broader testing)

---

## Future Investigation

### Questions Worth Testing

1. **Temporal Claims:** Does claim age correlate with source freshness patterns?
2. **Cross-language Patterns:** Do different Wikipedia languages show different source stability for contested topics?
3. **Domain Differences:** How do news articles, academic papers, and government sites differ in freshness patterns?
4. **Revert Patterns:** Are edit wars visible in revision comments, or do they manifest as subtle content drift?
5. **Authority Cascade:** Do tertiary sources citing secondary sources create evidence echo chambers?

### Data Collection Needed

- Longitudinal scans (same URL over time)
- Cross-domain comparisons (Wikipedia vs news vs academic)
- Multi-language Wikipedia comparisons
- Historical baseline (scan archived versions to measure drift)

---

## Disclaimers

These notes document observed patterns during tool development and testing. They are:
- **Not peer-reviewed research**
- **Not claims about truth or correctness**
- **Subject to revision with more data**
- **Limited by current test corpus**

For methodology details, see `docs/METHODOLOGY.md`.

For transparent scoring formulas, see source code in `internal/score/scorer.go`.
