"""Turns the message from Go into the list of numbers the model reads.

Go sends the same facts the formula uses (see MLFeatures in Go). Here we
add a few derived numbers that make the patterns easier to learn, for
example the money per person per day instead of the raw budget.
"""

# The order matters: the model was trained with the columns in this order.
FEATURE_NAMES = [
    "reputation_bp",       # creator's reputation, 0..10000
    "success_rate_bp",     # creator's success rate (5000 when no history)
    "finished",            # finished projects (completed + failed)
    "completed",           # successful projects
    "complexity",          # 1..10
    "account_age_days",    # capped at 365
    "team_size",
    "rub_per_person_day",  # budget in rubles / (team size x days)
    "duration_days",
    "deadline_ratio",      # days given / days needed (complexity x 7)
]

REQUIRED = [
    "budget", "duration_days", "complexity", "team_size", "reputation_bp",
    "success_rate_bp", "completed", "failed", "account_age_days",
]


def to_vector(f: dict) -> list[float]:
    """Check the fields and return the numbers in FEATURE_NAMES order."""
    missing = [k for k in REQUIRED if k not in f]
    if missing:
        raise ValueError(f"missing features: {', '.join(missing)}")
    for k in REQUIRED:
        if not isinstance(f[k], int) or isinstance(f[k], bool):
            raise ValueError(f"feature {k} must be an integer")

    completed = max(f["completed"], 0)
    finished = completed + max(f["failed"], 0)
    # Without history the success rate means nothing: use the neutral 50%,
    # the same idea as the formula in Go.
    success = f["success_rate_bp"] if finished > 0 else 5000
    team = max(f["team_size"], 1)
    days = max(f["duration_days"], 1)
    complexity = min(max(f["complexity"], 1), 10)
    rub_per_person_day = f["budget"] / 100 / (team * days)  # kopecks -> rubles

    return [
        float(min(max(f["reputation_bp"], 0), 10000)),
        float(success),
        float(finished),
        float(completed),
        float(complexity),
        float(min(max(f["account_age_days"], 0), 365)),
        float(team),
        float(rub_per_person_day),
        float(days),
        float(days / (complexity * 7)),
    ]