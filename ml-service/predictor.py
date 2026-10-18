"""Loads the trained model and turns features into a probability."""

from pathlib import Path

import joblib
import numpy as np

from features import FEATURE_NAMES, to_vector

DEFAULT_PATH = Path(__file__).parent / "models" / "model.joblib"


class Predictor:
    def __init__(self, model, version: str):
        self.model = model
        self.version = version

    @classmethod
    def load(cls, path: Path = DEFAULT_PATH) -> "Predictor":
        if not path.exists():
            raise FileNotFoundError(f"{path} not found: run  python train.py  first")
        bundle = joblib.load(path)
        if bundle["features"] != FEATURE_NAMES:
            raise ValueError("the model was trained with other features: run  python train.py  again")
        return cls(bundle["model"], bundle["version"])

    def predict_bp(self, features: dict) -> int:
        """Chance of success in basis points (10000 = 100%).

        Like the formula, it never says 0% or 100%: the result is kept
        between 5% and 95%.
        """
        x = np.array([to_vector(features)])
        p = float(self.model.predict_proba(x)[0][1])
        return round(min(max(p, 0.05), 0.95) * 10000)