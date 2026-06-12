import socket
import struct
import sys
import time

MCAST_GRP = '224.9.2.2'
MCAST_PORT = 208
IS_ALL_GROUPS = True

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM, socket.IPPROTO_UDP)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)

if IS_ALL_GROUPS:
    # on this port, collect ALL groups
    sock.bind(('', MCAST_PORT))
else:
    # on this port, collect GRP only
    sock.bind((MCAST_GRP, MCAST_PORT))

mreq = struct.pack("4sl", socket.inet_aton(MCAST_GRP), socket.INADDR_ANY)

sock.setsockopt(socket.IPPROTO_IP, socket.IP_ADD_MEMBERSHIP, mreq)

sock.settimeout(10.0)

print(f"Listening for multicast on {MCAST_GRP}:{MCAST_PORT}...")

try:
    data, addr = sock.recvfrom(10240)
    print(f"Received {len(data)} bytes from {addr}")
except socket.timeout:
    print("Timed out waiting for data.")
except Exception as e:
    print(f"Error: {e}")
finally:
    sock.close()
