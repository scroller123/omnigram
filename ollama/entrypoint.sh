#!/bin/sh
set -e

# Start Ollama server in the background
ollama serve &
OLLAMA_PID=$!

# Wait for Ollama to be ready
echo "Waiting for Ollama to start..."
until ollama list >/dev/null 2>&1; do
  sleep 1
done
echo "Ollama is ready."

# Create nomic-fast model if it doesn't already exist
if ! ollama list | grep -q "nomic-fast"; then
  echo "Pulling base model nomic-embed-text..."
  ollama pull nomic-embed-text

  echo "Creating nomic-fast model..."
  ollama create nomic-fast -f /modelfiles/Modelfile
  echo "nomic-fast model created successfully."
else
  echo "nomic-fast model already exists, skipping creation."
fi

# Wait for the Ollama server process
wait $OLLAMA_PID
