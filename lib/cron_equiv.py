"""Semantically-equivalent cron rewriter.

`rewrite(expr)` returns a DIFFERENT 5-field cron string that fires at exactly
the same times. GitHub re-attributes a scheduled workflow's actor only when the
`cron:` VALUE changes (comment/whitespace edits do not). So to re-home a cron
onto a durable account WITHOUT changing its schedule, we make a real but
schedule-neutral edit to the expression.

Strategy (first applicable wins):
  1. Expand a plain `*` field to its explicit full range
     (minute->0-59, hour->0-23, month->1-12, dom->1-31, dow->0-6).
  2. Reorder a comma list (`9,17` -> `17,9`); cron lists are unordered sets.
  3. Turn the first single numeric field into a one-element range (`4` -> `4-4`).

Safety: cron treats day-of-month and day-of-week as a UNION when BOTH are
restricted. Expanding `*` -> full-range on dom while dow is restricted (or
vice-versa) would silently widen the schedule (e.g. "Mondays" -> "daily"), so
those expansions are skipped unless the paired field is also `*`.
"""

# field index -> full-range replacement for a plain `*`
_FULL_RANGE = {
    0: "0-59",   # minute
    1: "0-23",   # hour
    2: "1-31",   # day-of-month
    3: "1-12",   # month
    4: "0-6",    # day-of-week
}

# Try always-safe fields first (minute, hour, month), then the dom/dow pair
# whose expansion is only conditionally safe.
_EXPAND_ORDER = [0, 1, 3, 2, 4]

_DOM, _DOW = 2, 4


def _expand_star(fields):
    """Expand the first safely-expandable plain `*` field. Returns True if done."""
    for i in _EXPAND_ORDER:
        if fields[i] != "*":
            continue
        if i == _DOM and fields[_DOW] != "*":
            continue  # would OR with a restricted dow -> widens schedule
        if i == _DOW and fields[_DOM] != "*":
            continue  # would OR with a restricted dom -> widens schedule
        fields[i] = _FULL_RANGE[i]
        return True
    return False


def _reorder_comma(fields):
    """Reverse the first comma list found. Returns True if a change was made."""
    for i, f in enumerate(fields):
        if "," in f:
            fields[i] = ",".join(reversed(f.split(",")))
            return True
    return False


def _single_value_range(fields):
    """Turn the first bare numeric field into an N-N range. True if changed."""
    for i, f in enumerate(fields):
        if f.isdigit():
            fields[i] = f"{f}-{f}"
            return True
    return False


def rewrite(expr):
    """Return a schedule-equivalent but textually different cron, or None.

    None means the input is not a 5-field cron, or no safe neutral edit exists.
    """
    if not expr:
        return None
    fields = expr.split()
    if len(fields) != 5:
        return None
    for transform in (_expand_star, _reorder_comma, _single_value_range):
        if transform(fields):
            return " ".join(fields)
    return None
