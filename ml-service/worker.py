"""The worker: reads requests from RabbitMQ, answers with the model's estimate.

Run:  python worker.py      (stop with Ctrl+C)
"""

import json
import logging
import os
import time

import pika
from dotenv import load_dotenv

from predictor import Predictor

# The same queue names as in Go (domain/prediction.go).
REQUESTS = "ml.predict"
RESULTS = "ml.result"

log = logging.getLogger("worker")


def handle_message(body: bytes, predictor: Predictor) -> dict:
    """Turn one request into one result. Raises ValueError for a bad message."""
    try:
        msg = json.loads(body)
        project_id = msg["project_id"]
        features = msg["features"]
    except (json.JSONDecodeError, KeyError, TypeError) as e:
        raise ValueError(f"bad message: {e}") from e
    if not isinstance(project_id, str) or not isinstance(features, dict):
        raise ValueError("bad message: wrong field types")

    return {
        "project_id": project_id,
        "probability_bp": predictor.predict_bp(features),
        "model_version": predictor.version,
    }


def run(url: str, predictor: Predictor) -> None:
    """One connection: consume until the connection breaks."""
    connection = pika.BlockingConnection(pika.URLParameters(url))
    channel = connection.channel()
    # durable=True: the same settings as in Go. Both sides may create the
    # queues; whoever starts first does it.
    channel.queue_declare(queue=REQUESTS, durable=True)
    channel.queue_declare(queue=RESULTS, durable=True)
    channel.basic_qos(prefetch_count=10)

    def on_message(ch, method, properties, body):
        try:
            result = handle_message(body, predictor)
        except ValueError as e:
            log.warning("dropped a message: %s", e)
            ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
            return
        ch.basic_publish(
            exchange="",
            routing_key=RESULTS,
            body=json.dumps(result).encode(),
            properties=pika.BasicProperties(content_type="application/json", delivery_mode=2),  # 2 = persistent
        )
        ch.basic_ack(delivery_tag=method.delivery_tag)
        log.info("project %s -> %d bp", result["project_id"], result["probability_bp"])

    channel.basic_consume(queue=REQUESTS, on_message_callback=on_message)
    log.info("connected to RabbitMQ, waiting for messages in %s", REQUESTS)
    channel.start_consuming()


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    logging.getLogger("pika").setLevel(logging.WARNING)  # pika's own messages are too detailed
    load_dotenv()  # reads ml-service/.env if it exists
    url = os.environ.get("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

    predictor = Predictor.load()
    log.info("model %s loaded", predictor.version)

    while True:  # reconnect forever, like the Go side
        try:
            run(url, predictor)
        except pika.exceptions.AMQPConnectionError as e:
            log.warning("cannot reach RabbitMQ (%s), retrying in 5s", e.__class__.__name__)
            time.sleep(5)
        except KeyboardInterrupt:
            log.info("stopped")
            return


if __name__ == "__main__":
    main()