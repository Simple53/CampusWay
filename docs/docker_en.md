## Run in Docker (CampusWay CLI)

Build the image from the repository root:

```bash
docker build -t campusway .
```

Run with your `config.toml` mounted:

```bash
docker run -d --name campusway \
  -v "$PWD/config.toml:/home/nonroot/config.toml" \
  -p 127.0.0.1:1080:1080 \
  -p 127.0.0.1:1081:1081 \
  --restart unless-stopped \
  campusway
```

Or use `docker-compose.yml` in the repository root:

```yaml
services:
  campusway:
    build:
      context: .
    image: campusway
    container_name: campusway
    restart: unless-stopped
    ports:
      - "127.0.0.1:1080:1080"
      - "127.0.0.1:1081:1081"
    volumes:
      - ./config.toml:/home/nonroot/config.toml
```

Start the service:

```bash
docker compose up -d
```

### aTrust Protocol Login and Data Persistence

The aTrust client needs to persist its state (`client_data.json`) across container restarts. Mount a Docker volume and configure `client_data_file` in `config.toml`:

```yaml
services:
  campusway:
    container_name: campusway
    image: campusway
    restart: unless-stopped
    ports:
      - "127.0.0.1:1080:1080"
      - "127.0.0.1:1081:1081"
    volumes:
      - ./config.toml:/home/nonroot/config.toml
      - campusway-data:/home/nonroot/data

volumes:
  campusway-data:
```

Set in `config.toml`:

```toml
protocol = "atrust"
client_data_file = "/home/nonroot/data/client_data.json"
```

When first logging in with aTrust where interactive verification is required, run once interactively:

```bash
docker compose run --rm -it campusway
```

Once `client_data.json` is saved, press `Ctrl+C` and start normally with `docker compose up -d`.
