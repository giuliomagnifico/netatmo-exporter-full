> ⚠️ **Warning**  
 
This forked version of the Netatmo exporter for Prometheus also works with [Thermostat](https://www.netatmo.com/en-eu/smart-thermostat). You need to compile it, I haven't made a Docker build. It exposes three metrics:

```
netatmo_thermostat_boiler_status
netatmo_thermostat_setpoint
netatmo_thermostat_temperature
```
Full overview:

```
# HELP netatmo_thermostat_boiler_status Netatmo Energy boiler status (1=on, 0=off). Per-room when possibile, otherwise per-home.
# TYPE netatmo_thermostat_boiler_status gauge
netatmo_thermostat_boiler_status{home_id="60796ad062axx",home_name="Casa",room_id="",room_name=""} 0
# HELP netatmo_thermostat_setpoint Netatmo Energy target setpoint temperature in degrees Celsius.
# TYPE netatmo_thermostat_setpoint gauge
netatmo_thermostat_setpoint{home_id="60796ad062xxx",home_name="Casa",room_id="3096xx",room_name=""} 0
# HELP netatmo_thermostat_temperature Netatmo Energy measured room temperature in degrees Celsius.
# TYPE netatmo_thermostat_temperature gauge
netatmo_thermostat_temperature{home_id="60796ad062axxx",home_name="Casa",room_id="30964xxx",room_name=""} 17.9
```

You need to compile it by downloading this repo and build it using

`go build -o netatmo-exporter`

Then, as before (see below), the original instructions apply: generate the token on the Netatmo dev website and grant the token **read thermostat** access.

Example in Grafana

<img width="1796" height="1568" alt="510765545-ce94b4f1-3b09-426e-8713-eb90682ed892" src="https://github.com/user-attachments/assets/50103055-3fd8-4804-9980-a86d192dd222" />


## Docker

It should works also using Docker, just download this repository:

`git clone https://github.com/giuliomagnifico/netatmo-exporter-full.git && cd netatmo-exporter-full`

Create the Docker image:

`docker build -t giuliomagnifico/netatmo-exporter-full:latest . `

And run it (follow also the original instruction below):

`docker volume create netatmo-token && docker run -d --name netatmo-exporter-full -p 9210:9210 -v netatmo-token:/var/lib/netatmo-exporter -e NETATMO_CLIENT_ID="xxx" -e NETATMO_CLIENT_SECRET="xxx" -e NETATMO_EXPORTER_EXTERNAL_URL="http://xxx:9210" giuliomagnifico/netatmo-exporter-full:latest`


---

**⬇️ Below the original instructions ⬇️**


___


# netatmo-exporter

Simple [prometheus](https://prometheus.io) exporter for getting sensor values [NetAtmo](https://www.netatmo.com) sensors into prometheus.

## Installation

### Run docker container

The exporter is available as a Docker image both on DockerHub and GitHub:

- [`ghcr.io/xperimental/netatmo-exporter`](https://github.com/xperimental/netatmo-exporter/pkgs/container/netatmo-exporter)
- [`xperimental/netatmo-exporter`](https://hub.docker.com/r/xperimental/netatmo-exporter/)

The following tags are available:

- `x.y.z` pointing to the release with that version
- `latest` pointing to the most recent released version
- `master` pointing to the latest build from the default branch

#### Token file and Docker volume

When running the `netatmo-exporter` in Docker, it is recommended to store the token file in a "Docker volume", so that it can persist container recreation. The image is already set up to do that. The default path for the token file is `/var/lib/netatmo-exporter/netatmo-token.json` and the whole `/var/lib/netatmo-exporter/` directory is set as a volume.

This enables the user to update the used netatmo-exporter image without losing the authentication, for example using `docker compose`. It does not automatically provide the same mechanism on Kubernetes, though. For Kubernetes, you probably want a `StatefulSet`.

### Build from source

Because this program uses the "Go Module" feature introduced in Go 1.11, you'll need at least that version of Go for building it.

If you have a working Go installation, getting the binary should be as simple as

```bash
git clone https://github.com/xperimental/netatmo-exporter
cd netatmo-exporter
make
```

If you want to build the exporter for a different OS or architecture, you can specify arguments to the Makefile:

```bash
# For 64-bit ARM on Linux
GOOS=linux GOARCH=arm64 make build-binary
```

## NetAtmo client credentials

This application tries to get data from the NetAtmo API. For that to work you will need to create an application in the [NetAtmo developer console](https://dev.netatmo.com/apps/), so that you can get a Client ID and secret.

For authentication, you either need to use the integrated web-interface of the exporter or you need to use the developer console to create a token and make manually make it available for the exporter to use. See [authentication.md](/doc/authentication.md) for more details.

The exporter is able to persist the authentication token during restarts, so that no user interaction is needed when restarting the exporter, unless the token expired during the time the exporter was not active. See [token-file.md](/doc/token-file.md) for an explanation of the file used for persisting the token.

## Usage

```plain
$ netatmo-exporter --help
Usage of netatmo-exporter:
  -a, --addr string                 Address to listen on. (default ":9210")
      --age-stale duration          Data age to consider as stale. Stale data does not create metrics anymore. (default 1h0m0s)
  -i, --client-id string            Client ID for NetAtmo app.
  -s, --client-secret string        Client secret for NetAtmo app.
      --debug-handlers              Enables debugging HTTP handlers.
      --external-url string         External URL to use as base for OAuth redirect URL.
      --log-level level             Sets the minimum level output through logging. (default info)
      --refresh-interval duration   Time interval used for internal caching of NetAtmo sensor data. (default 8m0s)
      --token-file string           Path to token file for loading/persisting authentication token.
```

After starting the server will offer the metrics on the `/metrics` endpoint, which can be used as a target for prometheus.

### Environment variables

The exporter can be configured either via command line arguments (see previous section) or by populating the following environment variables:

|                        Variable | Description                                                                |                                                   Default |
|--------------------------------:|----------------------------------------------------------------------------|----------------------------------------------------------:|
|         `NETATMO_EXPORTER_ADDR` | Address to listen on                                                       |                                                   `:9210` |
| `NETATMO_EXPORTER_EXTERNAL_URL` | External URL to use as base for OAuth redirect URL.                        |                                   `http://127.0.0.1:9210` |
|   `NETATMO_EXPORTER_TOKEN_FILE` | Path to token file for loading/persisting authentication token.            | (the Docker image has a default, which can be overridden) |
|                `DEBUG_HANDLERS` | Enables debugging HTTP handlers.                                           |                                                           |
|             `NETATMO_LOG_LEVEL` | Sets the minimum level output through logging.                             |                                                    `info` |
|      `NETATMO_REFRESH_INTERVAL` | Time interval used for internal caching of NetAtmo sensor data.            |                                                      `8m` |
|             `NETATMO_AGE_STALE` | Data age to consider as stale. Stale data does not create metrics anymore. |                                                      `1h` |
|             `NETATMO_CLIENT_ID` | Client ID for NetAtmo app.                                                 |                                                           |
|         `NETATMO_CLIENT_SECRET` | Client secret for NetAtmo app.                                             |                                                           |

### Cached data

The exporter has an in-memory cache for the data retrieved from the Netatmo API. The purpose of this is to decouple making requests to the Netatmo API from the scraping interval as the data from Netatmo does not update nearly as fast as the default scrape interval of Prometheus. Per the Netatmo documentation the sensor data is updated every ten minutes. The default "refresh interval" of the exporter is set a bit below this (8 minutes), but still much higher than the default Prometheus scrape interval (15 seconds).

You can still set a slower scrape interval for this exporter if you like:

```yml
scrape_configs:
  - job_name: 'netatmo'
    scrape_interval: 90s
    static_configs:
      - targets: ['localhost:9210']
```

### Troubleshooting

There have been issues with stale data in the NetAtmo account causing authentication issues. If you are getting `invalid_grant` errors when refreshing a token or the data refresh fails with an `Invalid access token` error then you might have this issue with your account.

In that case look at your [account page](https://home.netatmo.com/settings/my-account), navigate to the list of "Partner-Apps" and remove all entries related to the netatmo-exporter. The same option is also available in the mobile app.

Once this is done, remove the token file from the netatmo-exporter and re-authenticate.

## Links

- [Grafana Dashboard](https://grafana.com/grafana/dashboards/13672) contributed by [@GordonFreemanK](https://github.com/GordonFreemanK)
