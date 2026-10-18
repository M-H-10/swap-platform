"""Synthetic (invented) training data.

The platform is new, so there is no history of real projects yet. To train
and test the whole pipeline we generate projects at random and decide their
result with a hidden rule plus randomness. The rule is close to the
formula in Go, but it also has two effects the formula does not know:

  * a hard project (complexity 8+) with a team of 1-2 people fails more;
  * an experienced creator (reputation 7000+) handles hard projects better.

The model is never shown the rule, only the examples. If it learns these
effects from the examples, its estimates differ from the formula exactly
where the formula is too simple. When real projects exist, the same code
can be trained on them instead.
"""

import numpy as np

from features import FEATURE_NAMES, to_vector

# Column positions, to read the matrix by name.
COL = {name: i for i, name in enumerate(FEATURE_NAMES)}


def random_project(rng: np.random.Generator) -> dict:
    """One project in the same format Go sends."""
    finished = int(rng.poisson(3))
    skill = rng.beta(4, 2)  # the creator's hidden talent
    completed = int(rng.binomial(finished, skill))
    failed = finished - completed
    team = int(min(1 + rng.geometric(0.25), 40))
    days = int(rng.integers(7, 366))
    rub_per_person_day = float(rng.lognormal(np.log(1500), 0.8))
    return {
        "budget": max(int(rub_per_person_day * team * days * 100), 100_00),
        "duration_days": days,
        "complexity": int(rng.integers(1, 11)),
        "team_size": team,
        "reputation_bp": int(np.clip(rng.normal(5500, 1500), 0, 10000)),
        "success_rate_bp": completed * 10000 // finished if finished else 0,
        "completed": completed,
        "failed": failed,
        "account_age_days": int(rng.exponential(150)),
    }


def true_probability(X: np.ndarray) -> np.ndarray:
    """The hidden rule: the real chance of success of every row."""
    rep = X[:, COL["reputation_bp"]]
    success = X[:, COL["success_rate_bp"]]
    finished = X[:, COL["finished"]]
    completed = X[:, COL["completed"]]
    complexity = X[:, COL["complexity"]]
    age = X[:, COL["account_age_days"]]
    team = X[:, COL["team_size"]]
    rub = X[:, COL["rub_per_person_day"]]
    days = X[:, COL["duration_days"]]
    ratio = X[:, COL["deadline_ratio"]]

    z = (
        0.0003 * (rep - 5000)
        + 0.00025 * (success - 5000) * (finished > 0)
        + 0.06 * np.minimum(completed, 20)
        - 0.3 * (complexity - 5)
        - 1.0 * (age < 7)
        - 0.4 * ((age >= 7) & (age < 30))
        - 0.05 * np.maximum(team - 10, 0)
        + 0.7 * np.clip(np.log(rub / 1000), -2, 1.5)
        + np.where(ratio < 0.5, -1.3, np.where(ratio < 1, -0.5, np.where(days > 365, -0.3, 0.3)))
        # the two effects the formula does not know:
        - 0.8 * ((complexity >= 8) & (team <= 2))
        + 0.4 * ((rep > 7000) & (complexity >= 7))
    )
    return 1 / (1 + np.exp(-z))


def make_dataset(n: int = 20000, seed: int = 42) -> tuple[np.ndarray, np.ndarray]:
    """n projects (X) and their results (y: 1 = success, 0 = failure).

    The same seed always gives the same data, so training is repeatable.
    """
    rng = np.random.default_rng(seed)
    X = np.array([to_vector(random_project(rng)) for _ in range(n)])
    y = (rng.random(n) < true_probability(X)).astype(int)
    return X, y