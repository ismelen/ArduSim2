import socket
import json
import time
import sys
import random
import os

BROKER_IP = os.environ.get("BROKER_IP", "broker")
BROKER_PORT = int(os.environ.get("BROKER_PORT", "3000"))
UAV_ID = os.environ.get("UAV_ID", socket.gethostname())
EMIT_INTERVAL = float(os.environ.get("EMIT_INTERVAL", "0.5"))

def clamp_pos(val, min_val, max_val):
    return max(min_val, min(val, max_val))

def main():
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    
    # Random initial position in a 1km x 1km local map
    x = random.uniform(0.0, 1000.0)
    y = random.uniform(0.0, 1000.0)
    z = random.uniform(10.0, 50.0)

    # Register UAV
    pos_msg = {
        "topic": "telemetry",
        "payload": {
            "uav_id": UAV_ID,
            "payload": {
                "nr_gps_online": 10,
                "position": {
                    "heading": 0.0,
                    "alt": z,
                    "relative_alt": z,
                    "lon": x / 111120.0,
                    "lat": y / 111120.0
                },
                "type": "MAV_TYPE_QUADROTOR",
                "battery": 100,
                "version": "1.0",
                "time_boot_ms": int(time.time() * 1000),
                "speed": {"vx": 0.0, "vy": 0.0, "vz": 0.0},
                "status": "OK",
                "flight_mode": "GUIDED"
            }
        }
    }
    
    sock.sendto(json.dumps(pos_msg).encode('utf-8'), (BROKER_IP, BROKER_PORT))
    print(f"[{UAV_ID}] Registered at ({x:.4f}, {y:.4f}, {z:.1f})")
    
    time.sleep(1.0)
    
    messages_sent = 0
    for i in range(2):
        msg = {
            "topic": "broadcast",
            "payload": {
                "uav_id": UAV_ID,
                "payload": f"Test message {i} from {UAV_ID}"
            }
        }
        sock.sendto(json.dumps(msg).encode('utf-8'), (BROKER_IP, BROKER_PORT))
        messages_sent += 1
        time.sleep(0.5)
            
    print(f"[{UAV_ID}] Simulation ended. Sent {messages_sent} broadcast messages.")

if __name__ == "__main__":
    main()
