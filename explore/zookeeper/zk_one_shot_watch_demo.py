import os
import threading
import time

from kazoo.client import KazooClient


HOSTS = "127.0.0.1:2181"
ROOT = f"/watch-demo-{os.getpid()}"
CONFIG = f"{ROOT}/active-config"


def publish_updates(client):
    """Simulate a deployment service publishing two configuration versions."""
    time.sleep(1)
    for value in (b"version-2", b"version-3"):
        client.set(CONFIG, value)
        print(f"publisher wrote: {value.decode()}")
        time.sleep(1)


def main():
    publisher = KazooClient(hosts=HOSTS)
    subscriber = KazooClient(hosts=HOSTS)
    notification = threading.Event()
    callback_count = 0

    def config_changed(event):
        # This callback belongs to the subscriber session. Keep it non-blocking.
        nonlocal callback_count
        callback_count += 1
        print(f"subscriber callback: {event.type} on {event.path}")
        notification.set()

    publisher.start()
    subscriber.start()
    try:
        publisher.create(CONFIG, b"version-1", makepath=True)

        # A traditional watch is session-scoped and fires only once.
        value, _ = subscriber.get(CONFIG, watch=config_changed)
        print(f"subscriber initial read: {value.decode()}")

        thread = threading.Thread(target=publish_updates, args=(publisher,))
        thread.start()

        if not notification.wait(timeout=5):
            raise TimeoutError("watch callback did not fire")
        else:
            print("watch callback fired")

        # Re-read after the callback; the event only says that something changed.
        value, _ = subscriber.get(CONFIG)
        print(f"subscriber re-read: {value.decode()}")

        thread.join()
        print(f"callback count after two updates: {callback_count} (one-shot)")
    finally:
        publisher.delete(ROOT, recursive=True)
        subscriber.stop()
        subscriber.close()
        publisher.stop()
        publisher.close()


if __name__ == "__main__":
    main()
