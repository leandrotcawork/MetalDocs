# Build:   docker build -t metaldocs/gotenberg:local docker/gotenberg
# Run:     docker run --name metaldocs-gotenberg -p 3000:3000 -d metaldocs/gotenberg:local
# Verify:  ./docker/gotenberg/verify-carlito.sh metaldocs-gotenberg

# MetalDocs Gotenberg image — adds Carlito font (metric-compatible with
# Calibri) to the official Gotenberg image so Chromium HTML→PDF rendering
# uses the same metrics as the client editor and docx.js output.

FROM gotenberg/gotenberg:8

USER root

RUN apt-get update \
 && apt-get install -y --no-install-recommends fonts-crosextra-carlito fonts-liberation \
 && fc-cache -f \
 && rm -rf /var/lib/apt/lists/*

USER gotenberg

# Expose default Gotenberg port
EXPOSE 3000
