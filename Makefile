# Variables
BACKEND_IMAGE_NAME = platform-backend
DEVICE_IMAGE_NAME = platform-device
# Build backend image
build-backend:
	docker build -t $(BACKEND_IMAGE_NAME) ./backend
# Build device image
build-device:
	docker build -t $(DEVICE_IMAGE_NAME) ./device-sim
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
all: build-backend build-device run-backend run-device

start:
	@echo "Starting backend and device containers..."
	@$(MAKE) build-backend build-device run-backend run-device
	@echo "All containers started successfully."