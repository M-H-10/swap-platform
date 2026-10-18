import json

import pytest

from predictor import Predictor
from train import train
from worker import handle_message

STRONG = {  # trusted creator, easy project, enough time and money
    "budget": 600_000_00, "duration_days": 90, "complexity": 3, "team_size": 3,
    "reputation_bp": 8500, "success_rate_bp": 9000, "completed": 9, "failed": 1,
    "account_age_days": 300,
}
WEAK = {  # brand new account, very hard project, one person, a rushed deadline
    "budget": 20_000_00, "duration_days": 14, "complexity": 9, "team_size": 1,
    "reputation_bp": 5000, "success_rate_bp": 0, "completed": 0, "failed": 0,
    "account_age_days": 1,
}


@pytest.fixture(scope="module")
def predictor():
    # A small, fast model is enough to test the behaviour.
    model, metrics = train(n=4000, n_estimators=40)
    assert metrics["roc_auc"] > 0.7  # clearly better than guessing (0.5)
    return Predictor(model, "test")


def test_probability_is_between_5_and_95_percent(predictor):
    for project in (STRONG, WEAK):
        assert 500 <= predictor.predict_bp(project) <= 9500


def test_strong_project_beats_weak_project(predictor):
    assert predictor.predict_bp(STRONG) > predictor.predict_bp(WEAK) + 2000


def test_handle_message(predictor):
    body = json.dumps({"project_id": "abc", "features": STRONG}).encode()
    result = handle_message(body, predictor)
    assert result["project_id"] == "abc"
    assert result["model_version"] == "test"
    assert isinstance(result["probability_bp"], int)


@pytest.mark.parametrize("body", [b"not json", b"{}", b'{"project_id": 5, "features": {}}'])
def test_bad_messages_are_rejected(predictor, body):
    with pytest.raises(ValueError):
        handle_message(body, predictor)