import uuid
import requests
import time

DEVICE_ID = str(uuid.uuid4())
BACKEND_URL = "http://backend:8080/telemetry"
OTA_MANIFEST_URL = "http://ota-server:8081/manifest"
OTA_DOWNLOAD_TIMEOUT = 5

current_version = "1.0.0"


def send_telemetry():
    payload = {"device_id": DEVICE_ID, "message": "heartbeat"}
    try:
        resp = requests.post(BACKEND_URL, json=payload, timeout=2)
        print(f"Telemetry sent: {payload}, Response: {resp.status_code}", flush=True)
    except Exception as e:
        print(f"Failed to send telemetry: {e}", flush=True)


def check_for_ota_update():
    global current_version

    try:
        resp = requests.get(OTA_MANIFEST_URL, timeout=OTA_DOWNLOAD_TIMEOUT)
        resp.raise_for_status()
        manifest = resp.json()
    except Exception as e:
        print(f"Failed to check for OTA update: {e}", flush=True)
        return
    available_version = manifest.get("version")
    firmware_url = manifest.get("url")
    signature = manifest.get("signature")

    if not signature:
        print("OTA manifest missing signature, ignoring update", flush=True)
        return

    if not available_version or not firmware_url:
        print("Invalid OTA manifest received", flush=True)
        return
    if available_version == current_version:
        print(f"No update available, current version: {current_version}", flush=True)
        return
    print(
        f"New firmware version {available_version} available at {firmware_url}",
        flush=True,
    )
    download_and_apply_update(firmware_url, available_version, signature)


def download_and_apply_update(url: str, version: str, signature: str):
    global current_version

    try:
        resp = requests.get(url, timeout=OTA_DOWNLOAD_TIMEOUT)
        resp.raise_for_status()
        data = resp.content
        print(
            f"Downloaded firmware version {version}, size: {len(data)} bytes",
            flush=True,
        )
    except Exception as e:
        print(f"Failed to download firmware: {e}", flush=True)
        return

    if signature != "2dc9e2c8-c63c-441b-b805-ecfd5624f8ed":
        print("Firmware signature verification failed, update aborted", flush=True)
        return

    print(
        f"Applying firmware update to version {version} with signature {signature}...",
        flush=True,
    )
    current_version = version


def main_loop():
    print("Starting device simulator main loop", flush=True)
    counter = 0
    while True:
        send_telemetry()
        counter += 1
        if counter % 3 == 0:  # Check for OTA update every minute (
            check_for_ota_update()
        time.sleep(5)  # Send telemetry every 5 seconds


if __name__ == "__main__":
    main_loop()
