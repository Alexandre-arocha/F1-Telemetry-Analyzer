FROM python:3.12-slim
WORKDIR /app
COPY ingest/requirements.txt ingest/requirements.txt
RUN pip install --no-cache-dir -r ingest/requirements.txt
COPY ingest ingest
ENV INGEST_CACHE=/cache
ENTRYPOINT ["python", "-m", "ingest"]
