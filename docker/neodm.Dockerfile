FROM python:3.13-slim

WORKDIR /app

COPY middleware/services/neodm/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY middleware/services/neodm/ .
COPY middleware/proto/v1/gen/python/ /proto/v1/gen/python/

ENV PYTHONPATH=/proto/v1/gen/python:/app/proto/python

EXPOSE 50051
CMD ["python3", "main.py"]