#!/usr/bin/env python3
"""
Simulated drone client for network_module broker load testing.

Each drone:
  1. Sends 2 position updates, one per second
  2. Sends 2 random messages to random other drones (via the broker)
  3. Exits
"""

import json
import os
import random
import socket
import time

BROKER_HOST = os.environ.get("BROKER_HOST", "broker")
BROKER_PORT = int(os.environ.get("BROKER_PORT", "3000"))
TOTAL_DRONES = int(os.environ.get("TOTAL_DRONES", "100"))


def get_drone_id() -> str:
    """Derive a unique drone ID from the container hostname.
    Docker compose replicas produce hostnames like 'test-drones-1'.
    """
    if "DRONE_ID" in os.environ:
        return os.environ["DRONE_ID"]
    hostname = socket.gethostname()
    # Extract the last numeric part from the hostname
    parts = hostname.split("-")
    for part in reversed(parts):
        if part.isdigit():
            return f"drone_{int(part) - 1}"  # 0-indexed
    return f"drone_{random.randint(0, TOTAL_DRONES - 1)}"


DRONE_ID = get_drone_id()

# Random position in a ~1km area near Madrid
BASE_LAT = 40.4168
BASE_LON = -3.7038


def make_position():
    """Generate a random position with small drift."""
    return {
        "x": BASE_LAT + random.uniform(-0.005, 0.005),
        "y": BASE_LON + random.uniform(-0.005, 0.005),
        "z": random.uniform(0.05, 0.2),  # altitude in km-scale coords
    }


def send_udp(sock: socket.socket, data: dict):
    """Send a JSON UDP datagram to the broker."""
    raw = json.dumps(data).encode("utf-8")
    sock.sendto(raw, (BROKER_HOST, BROKER_PORT))


def pick_random_target(own_id: str, total: int) -> str:
    """Pick a random drone ID different from own_id."""
    while True:
        n = random.randint(0, total - 1)
        target = f"drone_{n}"
        if target != own_id:
            return target


def main():
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

    # --- Phase 1: send position update ---
    pos = make_position()
    msg = {
        "sender_id": DRONE_ID,
        "position": pos,
    }
    send_udp(sock, msg)
    print(f"[{DRONE_ID}] Position sent: ({pos['x']:.4f}, {pos['y']:.4f}, {pos['z']:.4f})")

    # --- Phase 2: wait for all drones to register, then send one message ---
    wait_secs = max(10, TOTAL_DRONES // 5)
    print(f"[{DRONE_ID}] Waiting {wait_secs}s for all drones to register...")
    time.sleep(wait_secs)

    target = pick_random_target(DRONE_ID, TOTAL_DRONES)
    payload = list(f"Hello from {DRONE_ID}".encode("utf-8"))
    msg = {
        "sender_id": DRONE_ID,
        "target_id": target,
        "payload": payload,
    }
    send_udp(sock, msg)
    print(f"[{DRONE_ID}] Message sent to {target}")

    sock.close()
    print(f"[{DRONE_ID}] Done.")


if __name__ == "__main__":
    main()
