import uuid
import requests
import time

DEVICE_ID = str(uuid.uuid4())
BACKEND_URL = "http://backend:8080/telemetry"


def send_telemetry():
    payload = {"device_id": DEVICE_ID, "message": "heartbeat"}
    try:
        resp = requests.post(BACKEND_URL, json=payload, timeout=2)
        print(f"Telemetry sent: {payload}, Response: {resp.status_code}", flush=True)
    except Exception as e:
        print(f"Failed to send telemetry: {e}", flush=True)


if __name__ == "__main__":
    print(f"Starting device simulator with ID: {DEVICE_ID}", flush=True)
    while True:
        send_telemetry()
        time.sleep(5)  # Send telemetry every 5 seconds
