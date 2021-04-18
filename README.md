# loki-docker-sd

**Use Loki with Docker without any configuration! (mostly)**

A service discovery mechanism for [Grafana Loki](https://grafana.com/oss/loki)
(actually Promtail) that discovers targets (log files) directly from a running
Docker daemon

## Why

Docker capabilities of upstream Loki may be insufficient, as they require
changing the Docker daemon configuration (`/etc/docker/daemon.json`) or setting
`--log-opts` on every container, which may not be desired or even impossible
(e.g. in provided / managed environments):

- [**Loki Docker logging
  driver**](https://grafana.com/docs/loki/latest/clients/docker-driver/)
  requires installing a Docker plugin on every machine the Docker daemon runs
  on, also `--log-driver=loki` must be passed to every `docker run`. Very tedious
- **[Others](https://gist.github.com/ruanbekker/c6fa9bc6882e6f324b4319c5e3622460)**
  suggest using `--log-opt tag=...` to "hook in" container information and pass
  that out using relabel rules. Again, requires setting something on every
  container (or in `/daemon.json`), which is tedious.
  
The Kubernetes capability of Loki however do not face this problem, as
`kubernetes_sd` hands Promtail the container ID, which can be used to grab the
correct logs from `/var/log/docker/containers`, while also retaining lots of
`__meta_` labels for identifying logs.

But Loki knows no `docker_sd` so we are out of luck? No!

Docker knows `file_sd` and the we can `docker ls` containers to get their ID.
And it gets even better, `docker inspect` even tells us `LogPath` – the absolute
path to the JSON file the containers logs are in.

`loki-docker-sd` leverages that to write targets to a JSON file suitable for
`file_sd` and sets `__path__` to `LogPath` so that Promtail can easily tail
our containers logs.

## Running

There is a docker container at `shorez/loki-docker-sd`. Requires access to `/var/run/docker.sock`:

```bash
$ docker run -v /var/run/docker.sock:/var/run/docker.sock:ro shorez/loki-docker-sd
```

For a complete example, see [`docker-compose.yml`](./docker-compose.yml)

## Configuration

> You said without configuration. LIAR

Yeah. lol. No Docker configuration at least and that's what counts.

A bit of Promtail configuration is still required (but that's not an issue). See
[`example-config.yml`](./example-config.yml) for that

### Meta labels
`loki-docker-sd` provides the following labels during relabeling:

- `__meta_docker_container_id`: Docker container ID
- `__meta_docker_container_name`: Docker container name
- `__meta_docker_container_status`: Docker container status (`running`, etc.)
- `__meta_docker_container_label_<label>`: Docker container labels (`.` and `-` are substituted by `_`)
- `__path__`: Path on disk to the log file
