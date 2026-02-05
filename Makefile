# Variables
BACKEND_IMAGE_NAME = platform-backend
DEVICE_IMAGE_NAME = platform-device
OTA_IMAGE_NAME = platform-ota
# Build backend image
build-backend:
	docker build -t $(BACKEND_IMAGE_NAME) ./backend
# Build device image
build-device:
	docker build -t $(DEVICE_IMAGE_NAME) ./device-sim
build-ota:
	docker build -t $(OTA_IMAGE_NAME) ./ota-server
# Run backend container
run-backend:
	docker run --rm -p 8080:8080 --name platform-backend $(BACKEND_IMAGE_NAME)
# Run device container
run-device:
	docker run --rm --name platform-device --link platform-backend:backend $(DEVICE_IMAGE_NAME)
# Stop and remove containers
clean:
	docker rm -f platform-backend platform-device || true

# Build and run everything
all: build-backend build-device build-ota run-backend run-device run-ota

start:
	@echo "Starting backend and device containers..."
	@$(MAKE) build-backend build-device build-ota run-backend run-device run-ota
	@echo "All containers started successfully."
