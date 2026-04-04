import socket
import json
import time

PORT = 3000
ADDR = ('127.0.0.1', PORT)

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

def send(data):
    sock.sendto(json.dumps(data).encode(), ADDR)

print("Registering drone_1 at (0,0)...")
send({
    "topic": "telemetry",
    "payload": {
        "uav_id": "drone_1",
        "payload": {
            "nr_gps_online": 10,
            "position": {"lat": 0.0, "lon": 0.0, "alt": 10.0, "relative_alt": 10.0, "heading": 0.0},
            "speed": {"vx": 0.0, "vy": 0.0, "vz": 0.0},
            "battery": 100,
            "type": "copter",
            "flight_mode": "GUIDED",
            "status": "ACTIVE",
            "time_boot_ms": 1000,
            "version": "1.0"
        }
    }
})

print("Registering drone_2 at (1,1) - Out of range...")
send({
    "topic": "telemetry",
    "payload": {
        "uav_id": "drone_2",
        "payload": {
            "nr_gps_online": 10,
            "position": {"lat": 1.0, "lon": 1.0, "alt": 10.0, "relative_alt": 10.0, "heading": 0.0},
            "speed": {"vx": 0.0, "vy": 0.0, "vz": 0.0},
            "battery": 100,
            "type": "copter",
            "flight_mode": "GUIDED",
            "status": "ACTIVE",
            "time_boot_ms": 1000,
            "version": "1.0"
        }
    }
})

time.sleep(1)

print("\n--- Test 1: Local Broadcast from drone_1 ---")
print("Expected: drone_2 should NOT receive (discarded: distance loss)")
send({
    "topic": "broadcast",
    "payload": {
        "uav_id": "drone_1",
        "payload": [1, 2, 3]
    }
})

time.sleep(1)

print("\n--- Test 2: Global Broadcast (uav_id empty) ---")
print("Expected: Both should receive (SUCCESS)")
send({
    "topic": "broadcast",
    "payload": {
        "uav_id": "",
        "payload": [4, 5, 6]
    }
})

time.sleep(1)
print("\nVerification script finished.")
