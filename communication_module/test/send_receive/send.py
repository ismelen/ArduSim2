import socket
import time
import random 
import sys
import json

def createTimeMessage():
    current_time_millis = int(time.time() * 1000)
    message = {"topic": "timing", "sender":socket.gethostname(), "time": current_time_millis}
    json_bytes = json.dumps(message).encode('utf-8')
    return json_bytes

def getRandomMessage():
    data = [
        b"""
        {
          "topic": "receive",
          "payload": "hello world"
        }""",
        b"""
        {
          "topic": "other_topic",
          "payload": "test test"
        }"""
    ]
    message = random.choice(data)
    return message

HOST_IP = "0.0.0.0"
HOST_PORT = 3000

try:
    BROKER_IP = sys.argv[1]
    BROKER_PORT = int(sys.argv[2])
except ValueError:
    print("Error: Second argument must be an integer.",flush=True)
    sys.exit(1)

random.seed()
# Create a UDP socket
udp_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

# Bind the socket to a specific address and port to receive data
local_address = (HOST_IP, HOST_PORT)

# Send data to a specific IP address and port
remote_address = (BROKER_IP, BROKER_PORT)

print("start sending messages",flush=True)
while True:
    message = createTimeMessage()
    udp_socket.sendto(message, remote_address)
    #print("send ", message.decode(),flush=True)

    # Receive data from any IP address and port
    time.sleep(random.randint(0,5))

# Close the socket when finished
udp_socket.close()

