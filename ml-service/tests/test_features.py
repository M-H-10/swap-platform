import pytest

from features import FEATURE_NAMES, to_vector

PROJECT = {
    "budget": 100_000_00,  # 100 000 rubles in kopecks
    "duration_days": 30,
    "complexity": 5,
    "team_size": 2,
    "reputation_bp": 6000,
    "success_rate_bp": 7500,
    "completed": 3,
    "failed": 1,
    "account_age_days": 400,
}


def test_vector_values():
    v = dict(zip(FEATURE_NAMES, to_vector(PROJECT)))
    assert v["finished"] == 4
    assert v["success_rate_bp"] == 7500
    assert v["account_age_days"] == 365                      # capped
    assert v["rub_per_person_day"] == pytest.approx(100_000 / 60)
    assert v["deadline_ratio"] == pytest.approx(30 / 35)


def test_no_history_means_neutral_success_rate():
    v = dict(zip(FEATURE_NAMES, to_vector({**PROJECT, "completed": 0, "failed": 0, "success_rate_bp": 0})))
    assert v["success_rate_bp"] == 5000


def test_missing_feature_is_rejected():
    bad = {k: v for k, v in PROJECT.items() if k != "budget"}
    with pytest.raises(ValueError, match="budget"):
        to_vector(bad)


def test_text_instead_of_number_is_rejected():
    with pytest.raises(ValueError, match="complexity"):
        to_vector({**PROJECT, "complexity": "5"})