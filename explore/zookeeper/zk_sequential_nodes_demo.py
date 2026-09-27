import os
import threading

from kazoo.client import KazooClient


HOSTS = "127.0.0.1:2181"
ROOT = f"/sequential-demo-{os.getpid()}"
ORDERS = f"{ROOT}/orders"


def submit_order(client, customer):
    path = client.create(
        f"{ORDERS}/order-",
        customer.encode(),
        sequence=True,
    )
    print(f"{customer} received name: {path.rsplit('/', 1)[1]}")

def print_tree(zk, path="/", indent=0):
    """
    Recursively walks through the ZooKeeper hierarchy and prints it like a tree.
    """

    try:
        # Get data value and metadata stats for the current path
        data, stat = zk.get(path)

        # Skip displaying value for the absolute root to keep output clean
        val_str = f" -> value: {data.decode('utf-8', errors='ignore')}" if path != "/" else ""

        # Extract the node name from the full path
        node_name = path.split("/")[-1] if path != "/" else "/"

        # Print with visual indentation indentation
        print("  " * indent + f"📁 {node_name} (v{stat.version}){val_str}")

        # Fetch children paths and sort them alphabetically
        children = sorted(zk.get_children(path))

        for child in children:
            # Build the fully qualified path for the child node
            child_path = f"{path}/{child}" if path != "/" else f"/{child}"

            # Skip ZK's internal configuration system node to keep the tree relevant
            if child_path == "/zookeeper":
                continue

            # Recursive call for sub-nodes
            print_tree(zk, child_path, indent + 1)

    except Exception as e:
        print("  " * indent + f"❌ Error reading {path}: {e}")

def main():
    alice = KazooClient(hosts=HOSTS)
    bob = KazooClient(hosts=HOSTS)
    alice.start()
    bob.start()
    try:
        alice.ensure_path(ORDERS)

        # Two independent sessions submit concurrently. ZooKeeper assigns the
        # monotonically increasing suffixes that establish their total order.
        threads = [
            threading.Thread(target=submit_order, args=(alice, "alice")),
            threading.Thread(target=submit_order, args=(bob, "bob")),
        ]
        for thread in threads:
            thread.start()
        for thread in threads:
            thread.join()

        print("\nOrders in ZooKeeper order:")
        print_tree(alice, ROOT)
        for child in sorted(alice.get_children(ORDERS)):
            data, _ = alice.get(f"{ORDERS}/{child}")
            print(f"{child} -> {data.decode()}")
    finally:
        alice.delete(ROOT, recursive=True)
        bob.stop()
        bob.close()
        alice.stop()
        alice.close()


if __name__ == "__main__":
    main()
