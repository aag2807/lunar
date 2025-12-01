#!/usr/bin/env python3
import json
import subprocess
import sys

def send_message(proc, method, params, msg_id=None):
    """Send an LSP message"""
    message = {
        "jsonrpc": "2.0",
        "method": method,
        "params": params
    }
    if msg_id is not None:
        message["id"] = msg_id

    content = json.dumps(message)
    header = f"Content-Length: {len(content)}\r\n\r\n"

    proc.stdin.write(header.encode())
    proc.stdin.write(content.encode())
    proc.stdin.flush()
    print(f"→ Sent: {method}", file=sys.stderr)

def read_response(proc):
    """Read an LSP response"""
    # Read Content-Length header
    line = proc.stdout.readline().decode()
    if not line.startswith("Content-Length:"):
        return None

    length = int(line.split(":")[1].strip())

    # Read empty line
    proc.stdout.readline()

    # Read content
    content = proc.stdout.read(length).decode()
    return json.loads(content)

def main():
    lsp_path = r"C:\Users\Alvaro\Development\lua\lunar\bin\lunar-lsp.exe"

    print("Starting LSP server...", file=sys.stderr)
    proc = subprocess.Popen(
        [lsp_path],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )

    try:
        # Initialize
        send_message(proc, "initialize", {
            "processId": None,
            "rootUri": "file:///C:/Users/Alvaro/Development/lua/portfolio",
            "capabilities": {}
        }, 1)

        init_resp = read_response(proc)
        print(f"← Init: {init_resp is not None}", file=sys.stderr)

        # Initialized notification
        send_message(proc, "initialized", {})

        # Open document with "local t: TEST" and then "t:save_me"
        send_message(proc, "textDocument/didOpen", {
            "textDocument": {
                "uri": "file:///C:/Users/Alvaro/Development/lua/portfolio/main.lunar",
                "languageId": "lunar",
                "version": 1,
                "text": "local t: TEST\nt:save_me"
            }
        })

        # Request completion after "t:save_me"
        send_message(proc, "textDocument/completion", {
            "textDocument": {
                "uri": "file:///C:/Users/Alvaro/Development/lua/portfolio/main.lunar"
            },
            "position": {
                "line": 1,  # Second line where "t:save_me" is
                "character": 9  # After "t:save_me"
            },
            "context": {
                "triggerKind": 1  # Invoked (manual trigger)
            }
        }, 2)

        # Read all responses (including diagnostics notifications)
        responses = []
        import time
        time.sleep(0.5)  # Give time for all responses

        # Try to read multiple responses
        for i in range(5):
            try:
                resp = read_response(proc)
                if resp:
                    responses.append(resp)
                    print(f"← Response {i+1}: method={resp.get('method', resp.get('id'))}", file=sys.stderr)
            except:
                break

        # Find completion response
        completion_resp = None
        for resp in responses:
            if resp.get("id") == 2:  # Our completion request ID
                completion_resp = resp
                break

        print("\n=== COMPLETION RESPONSE ===", file=sys.stderr)
        if completion_resp and "result" in completion_resp:
            items = completion_resp["result"].get("items", [])
            print(f"Found {len(items)} completion items:", file=sys.stderr)
            for item in items:
                print(f"  - {item.get('label')}: {item.get('detail', 'no detail')}", file=sys.stderr)
        else:
            print("No completion items found", file=sys.stderr)
            if completion_resp:
                print(json.dumps(completion_resp, indent=2))

        # Shutdown
        send_message(proc, "shutdown", {}, 3)
        send_message(proc, "exit", {})

        proc.wait(timeout=5)

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        proc.kill()
        raise
    finally:
        # Print stderr output
        stderr = proc.stderr.read().decode()
        if stderr:
            print("\n=== LSP STDERR ===", file=sys.stderr)
            print(stderr, file=sys.stderr)

if __name__ == "__main__":
    main()
