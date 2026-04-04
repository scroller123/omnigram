cat > /tmp/Modelfile <<EOF
FROM nomic-embed-text
PARAMETER num_ctx 1024
PARAMETER num_thread 4
EOF

ollama create nomic-fast -f /tmp/Modelfile

```
ollama show nomic-fast --parameters
```

cat > /tmp/Modelfile <<EOF
FROM nomic-embed-text
PARAMETER num_ctx 512
PARAMETER num_thread 4
EOF

ollama create nomic-fast-512 -f /tmp/Modelfile
