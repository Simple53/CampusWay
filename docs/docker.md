## Docker 运行 (CampusWay CLI)

首先在仓库根目录构建镜像：

```bash
docker build -t campusway .
```

然后挂载配置文件运行：

```bash
docker run -d --name campusway \
  -v "$PWD/config.toml:/home/nonroot/config.toml" \
  -p 127.0.0.1:1080:1080 \
  -p 127.0.0.1:1081:1081 \
  --restart unless-stopped \
  campusway
```

也可以使用 Docker Compose。直接编辑或参考仓库中的 `docker-compose.yml`：

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

在同目录下运行：

```bash
docker compose up -d
```

### aTrust 协议登录与数据持久化

aTrust 协议客户端需要在容器内持久化登录凭据（`client_data.json`），可参考以下示例将凭据保存在 Docker 卷中：

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

在 `config.toml` 中设置：

```toml
protocol = "atrust"
client_data_file = "/home/nonroot/data/client_data.json"
```

初次使用 aTrust 协议登录需要图形验证码或短信验证码交互时，可临时以交互模式运行：

```bash
docker compose run --rm -it campusway
```

登录成功并保存 `client_data.json` 后按 `Ctrl+C` 退出临时容器，再使用 `docker compose up -d` 后台常驻运行即可。
