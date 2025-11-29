import subprocess
import json
import sys
import os
import time

def rpc_request(process, method, params=None, req_id=1):
    msg = {
        "jsonrpc": "2.0",
        "method": method,
        "id": req_id
    }
    if params:
        msg["params"] = params
    
    json_msg = json.dumps(msg)
    print(f"Sending: {json_msg}", file=sys.stderr)
    process.stdin.write(json_msg + "\n")
    process.stdin.flush()

    while True:
        line = process.stdout.readline()
        if not line:
            return None
        # print(f"Received raw: {line.strip()}", file=sys.stderr)
        try:
            response = json.loads(line)
            if response.get("id") == req_id:
                return response
        except json.JSONDecodeError:
            continue

def main():
    project_path = "/Users/pelayo/projects/hexanorm"
    cmd = ["go", "run", ".", project_path]
    
    print(f"Starting server with: {cmd}", file=sys.stderr)
    process = subprocess.Popen(
        cmd,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=sys.stderr,
        text=True,
        cwd=project_path
    )

    try:
        # Initialize
        init_params = {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "antigravity-client", "version": "1.0"}
        }
        resp = rpc_request(process, "initialize", init_params, 1)
        print("Initialization response:", json.dumps(resp, indent=2))

        # Initialized notification
        notify_msg = {
            "jsonrpc": "2.0",
            "method": "notifications/initialized"
        }
        process.stdin.write(json.dumps(notify_msg) + "\n")
        process.stdin.flush()

        # List Tools
        resp = rpc_request(process, "tools/list", {}, 2)
        print("Tools list:", json.dumps(resp, indent=2))
        
        # List Resources
        resp = rpc_request(process, "resources/list", {}, 3)
        print("Resources list:", json.dumps(resp, indent=2))

        # Read Status Resource
        read_params = {"uri": "mcp://vibecoder/status"}
        resp = rpc_request(process, "resources/read", read_params, 4)
        print("Status resource:", json.dumps(resp, indent=2))

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
    finally:
        process.terminate()

if __name__ == "__main__":
    main()
