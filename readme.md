[![](https://img.shields.io/discord/828676951023550495?color=5865F2&logo=discord&logoColor=white)](https://discord.com/invite/yYd6YKHQZH)
![](https://img.shields.io/github/repo-size/shi-gghi-gg/mellow-a7s?maxAge=3600)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/I3I6AFVAP)

**⚠️ In development, breaking changes ⚠️**

## About
This is the analytics engine (v2) for the [wamellow.com](https://wamellow.com) discord bot, tracking command usage by command name, tracking and users.

If you need help using this, join **[our Discord Server](https://discord.com/invite/yYd6YKHQZH)**.

## Setup
Create the following `.env`:
```
CLICKHOUSE_USER=analytics_user
CLICKHOUSE_PASSWORD=your_super_secret_clickhouse_password

GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=your_grafana_admin_password

GRAFANA_ANONYMOUS_ACCESS_ENABLED=true
GRAFANA_ANONYMOUS_ORG_NAME=Wamellow
GRAFANA_ANONYMOUS_ORG_ROLE=Viewer

API_TOKEN=secure
```