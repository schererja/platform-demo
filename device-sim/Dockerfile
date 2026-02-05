FROM python:3.12-slim

WORKDIR /app
COPY device.py .

RUN pip install requests

ENTRYPOINT [ "python", "device.py" ]