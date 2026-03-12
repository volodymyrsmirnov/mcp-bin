#!/usr/bin/env python3
"""Simple MCP server for testing mcp-bin."""
import json
import sys

def send(msg):
    sys.stdout.write(json.dumps(msg) + "\n")
    sys.stdout.flush()

def main():
    for line in sys.stdin:
        msg = json.loads(line.strip())

        if msg.get("method") == "initialize":
            send({
                "jsonrpc": "2.0",
                "id": msg["id"],
                "result": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {"tools": {}},
                    "serverInfo": {"name": "test-server", "version": "1.0.0"}
                }
            })
        elif msg.get("method") == "notifications/initialized":
            pass  # notification, no response needed
        elif msg.get("method") == "tools/list":
            send({
                "jsonrpc": "2.0",
                "id": msg["id"],
                "result": {
                    "tools": [
                        {
                            "name": "greet",
                            "description": "Greet someone by name",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "name": {"type": "string", "description": "Name to greet"},
                                    "loud": {"type": "boolean", "description": "Use uppercase"}
                                },
                                "required": ["name"]
                            }
                        },
                        {
                            "name": "add",
                            "description": "Add two numbers",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "a": {"type": "number", "description": "First number"},
                                    "b": {"type": "number", "description": "Second number"}
                                },
                                "required": ["a", "b"]
                            }
                        }
                    ]
                }
            })
        elif msg.get("method") == "tools/call":
            name = msg["params"]["name"]
            args = msg["params"].get("arguments", {})

            if name == "greet":
                greeting = f"Hello, {args.get('name', 'World')}!"
                if args.get("loud"):
                    greeting = greeting.upper()
                send({
                    "jsonrpc": "2.0",
                    "id": msg["id"],
                    "result": {
                        "content": [{"type": "text", "text": greeting}]
                    }
                })
            elif name == "add":
                result = args.get("a", 0) + args.get("b", 0)
                send({
                    "jsonrpc": "2.0",
                    "id": msg["id"],
                    "result": {
                        "content": [{"type": "text", "text": str(result)}]
                    }
                })
            else:
                send({
                    "jsonrpc": "2.0",
                    "id": msg["id"],
                    "result": {
                        "content": [{"type": "text", "text": f"Unknown tool: {name}"}],
                        "isError": True
                    }
                })

if __name__ == "__main__":
    main()
