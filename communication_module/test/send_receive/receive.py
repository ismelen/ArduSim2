import socket
import time
import random 
import sys
import json

def getData(json_bytes):
    decoded_message = json.loads(json_bytes.decode('utf-8'))
    if decoded_message.get("topic") == "timing":
        message_time = decoded_message.get("time")
        current_time = int(time.time() * 1000)
        time_difference = current_time - message_time
        sender = decoded_message.get("sender")
        return time_difference,sender
    else:
        return None

HOST_IP = "0.0.0.0"
HOST_PORT = 3000

try:
    BROKER_IP = sys.argv[1]
    BROKER_PORT = int(sys.argv[2])
except ValueError:
    print("Error: Second argument must be an integer.",flush=True)
    sys.exit(1)


# Create a UDP socket
udp_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

# Bind the socket to a specific address and port to receive data
local_address = (HOST_IP, HOST_PORT)
udp_socket.bind(local_address)
udp_socket.settimeout(2.0)


# Send data to a specific IP address and port
remote_address = (BROKER_IP, BROKER_PORT)

message = subscribe = b"""
{
  "topic": "$subscribe",
  "subscribe_to": "timing"
}"""

udp_socket.sendto(message, remote_address)

while True:
    # Receive data from any IP address and port
    try:
        data, address = udp_socket.recvfrom(1024)
        time_difference,sender = getData(data)

        print(f"Time to receive from {sender} = {time_difference} ms",flush=True)
        #print(f"Received message from {address}: {data.decode()}",flush=True)
    except:
        pass
    

# Close the socket when finished
udp_socket.close()

