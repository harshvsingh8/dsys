import time
import traceback
from kazoo.client import KazooClient
from kazoo.exceptions import KazooException
from kazoo.exceptions import NodeExistsError

def test_add_node(zk):
    tag_str = str(time.time()).encode('utf-8')  
    try:
        zk.create("/my/hello", b"world@" + tag_str, makepath=True)
    except NodeExistsError:
        print("Node already exists. with value:", zk.get("/my/hello")[0])
        # Update the node with a new timestamped value
        zk.set("/my/hello", b"world@" + tag_str)
        pass
    except KazooException as e:
        print(f"An error occurred: {e}")
        traceback.print_exc()

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

if __name__ == "__main__":
    zk = KazooClient(hosts='127.0.0.1:2181', timeout=5.0)
    print("Starting Zookeeper client...")
    zk.start()
    print("Zookeeper client started.")
    print_tree(zk)
    test_add_node(zk)
    print_tree(zk)
    zk.stop()
    print("Zookeeper client stopped.")
