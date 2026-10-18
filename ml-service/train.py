"""Trains the Random Forest and saves it to models/model.joblib.

Run:  python train.py
"""

from pathlib import Path

import joblib
from sklearn.ensemble import RandomForestClassifier
from sklearn.metrics import accuracy_score, roc_auc_score
from sklearn.model_selection import train_test_split

from dataset import make_dataset
from features import FEATURE_NAMES

MODEL_VERSION = "rf-1"
MODEL_PATH = Path(__file__).parent / "models" / "model.joblib"


def train(n: int = 20000, n_estimators: int = 150, seed: int = 42):
    """Return the trained model and its scores on data it has never seen."""
    X, y = make_dataset(n, seed)
    # 80% to learn from, 20% kept aside to check the model honestly.
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=seed, stratify=y)

    model = RandomForestClassifier(
        n_estimators=n_estimators,  # number of trees that vote
        max_depth=10,               # how many questions one tree may ask
        min_samples_leaf=20,        # every answer must rest on 20+ examples
        n_jobs=-1,                  # use all CPU cores
        random_state=seed,
    )
    model.fit(X_train, y_train)

    proba = model.predict_proba(X_test)[:, 1]
    metrics = {
        "accuracy": accuracy_score(y_test, proba >= 0.5),
        "roc_auc": roc_auc_score(y_test, proba),
        "importance": dict(zip(FEATURE_NAMES, model.feature_importances_)),
    }
    return model, metrics


def main():
    print("training on 20000 synthetic projects...")
    model, m = train()
    print(f"accuracy on the test set: {m['accuracy']:.3f}")
    print(f"ROC AUC on the test set:  {m['roc_auc']:.3f}")
    print("what the model looks at most:")
    for name, value in sorted(m["importance"].items(), key=lambda kv: -kv[1]):
        print(f"  {name:20s} {value:.3f}")

    MODEL_PATH.parent.mkdir(exist_ok=True)
    joblib.dump({"model": model, "version": MODEL_VERSION, "features": FEATURE_NAMES},
                MODEL_PATH, compress=3)
    print(f"saved {MODEL_PATH} (version {MODEL_VERSION})")


if __name__ == "__main__":
    main()