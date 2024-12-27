# Proxmox Backup Server Exporter

[![license](https://img.shields.io/github/license/natrontech/pbs-exporter)](https://github.com/natrontech/pbs-exporter/blob/main/LICENSE)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/natrontech/pbs-exporter/badge)](https://securityscorecards.dev/viewer/?uri=github.com/natrontech/pbs-exporter)
[![release](https://img.shields.io/github/v/release/natrontech/pbs-exporter)](https://github.com/natrontech/pbs-exporter/releases)
[![go-version](https://img.shields.io/github/go-mod/go-version/natrontech/pbs-exporter)](https://github.com/natrontech/pbs-exporter/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/natrontech/pbs-exporter)](https://goreportcard.com/report/github.com/natrontech/pbs-exporter)
[![SLSA 3](https://slsa.dev/images/gh-badge-level3.svg)](https://slsa.dev)

---

Export [Proxmox Backup Server](https://www.proxmox.com/en/proxmox-backup-server/overview) statistics to [Prometheus](https://prometheus.io/).

Metrics are retrieved using the [Proxmox Backup Server API](https://pbs.proxmox.com/docs/api-viewer/index.html).

## Exported Metrics

| Metric                         | Meaning                                                 | Labels                                       |
| ------------------------------ | ------------------------------------------------------- | -------------------------------------------- |
| pbs_up                         | Was the last query of Proxmox Backup Server successful? |                                              |
| pbs_version                    | Version of Proxmox Backup Server                        | `version`, `repoid`, `release`               |
| pbs_available                  | The available bytes of the underlying storage.          | `datastore`                                  |
| pbs_size                       | The size of the underlying storage in bytes.            | `datastore`                                  |
| pbs_used                       | The used bytes of the underlying storage.               | `datastore`                                  |
| pbs_snapshot_count             | The total number of backups.                            | `datastore`, `namespace`                     |
| pbs_snapshot_vm_count          | The total number of backups per VM.                     | `datastore`, `namespace`, `vm_id`, `vm_name` |
| pbs_snapshot_vm_last_timestamp | The timestamp of the last backup of a VM.               | `datastore`, `namespace`, `vm_id`, `vm_name` |
| pbs_snapshot_vm_last_verify    | The verify status of the last backup of a VM.           | `datastore`, `namespace`, `vm_id`, `vm_name` |
| pbs_host_cpu_usage             | The CPU usage of the host.                              |                                              |
| pbs_host_memory_free           | The free memory of the host.                            |                                              |
| pbs_host_memory_total          | The total memory of the host.                           |                                              |
| pbs_host_memory_used           | The used memory of the host.                            |                                              |
| pbs_host_swap_free             | The free swap of the host.                              |                                              |
| pbs_host_swap_total            | The total swap of the host.                             |                                              |
| pbs_host_swap_used             | The used swap of the host.                              |                                              |
| pbs_host_disk_available        | The available disk of the local root disk in bytes.     |                                              |
| pbs_host_disk_total            | The total disk of the local root disk in bytes.         |                                              |
| pbs_host_disk_used             | The used disk of the local root disk in bytes.          |                                              |
| pbs_host_uptime                | The uptime of the host.                                 |                                              |
| pbs_host_io_wait               | The io wait of the host.                                |                                              |
| pbs_host_load1                 | The load for 1 minute of the host.                      |                                              |
| pbs_host_load5                 | The load for 5 minutes of the host.                     |                                              |
| pbs_host_load15                | The load 15 minutes of the host.                        |                                              |

## Flags / Environment Variables

```bash
$ ./pbs-exporter -help
```

You can use the following flags to configure the exporter. All flags can also be set using environment variables. Environment variables take precedence over flags.

| Flag                     | Environment Variable | Description                                          | Default                                                |
| ------------------------ | -------------------- | ---------------------------------------------------- | ------------------------------------------------------ |
| `pbs.loglevl`            | `PBS_LOGLEVEL`       | Log level (debug, info)                              | `info`                                                 |
| `pbs.api.token`          | `PBS_API_TOKEN`      | API token to use for authentication                  |                                                        |
| `pbs.api.token.name`     | `PBS_API_TOKEN_NAME` | Name of the API token to use for authentication      | `pbs-exporter`                                         |
| `pbs.endpoint`           | `PBS_ENDPOINT`       | Address of the Proxmox Backup Server                 | `http://localhost:8007` (if no parameter `target` set) |
| `pbs.username`           | `PBS_USERNAME`       | Username to use for authentication                   | `root@pam`                                             |
| `pbs.timeout`            | `PBS_TIMEOUT`        | Timeout for requests to Proxmox Backup Server        | `5s`                                                   |
| `pbs.insecure`           | `PBS_INSECURE`       | Disable TLS certificate verification                 | `false`                                                |
| `pbs.metrics-path`       | `PBS_METRICS_PATH`   | Path under which to expose metrics                   | `/metrics`                                             |
| `pbs.web.listen-address` | `PBS_LISTEN_ADDRESS` | Address to listen on for web interface and telemetry | `:10019`                                                |

### Docker secrets

If you are using [Docker secrets](https://docs.docker.com/engine/swarm/secrets/), you can use the following environment variables to set the path to the secrets:

| Environment Variable      | Description                     |
| ------------------------- | ------------------------------- |
| `PBS_API_TOKEN_FILE`      | Path to the API token file      |
| `PBS_API_TOKEN_NAME_FILE` | Path to the API token name file |
| `PBS_USERNAME_FILE`       | Path to the username file       |

See an example of how to use Docker secrets with Docker Compose in the [docker-compose-secrets.yaml](docker-compose-secrets.yaml) file.

The variables `PBS_API_TOKEN`, `PBS_API_TOKEN_NAME`, and `PBS_USERNAME` take precedence over the secret files.

## Multiple Proxmox Backup Servers

If you want to monitor multiple Proxmox Backup Servers, you can use the `targets` parameter in the query string. Instead of setting the `pbs.endpoint` flag (or `PBS_ENDPOINT` env), you can use the `target` parameter in the query string to specify the Proxmox Backup Server to monitor. You would then use following URL to scrape metrics: `http://localhost:10019/metrics?target=http://10.10.10.10:8007`.

This is useful if you are using Prometheus and want to monitor multiple Proxmox Backup Servers with one "pbs-exporter" instance.
You find examples for Prometheus static configuration in the [prometheus/static-config](prometheus/static-config) directory.

:warning: **Important**: if `pbs.endpoint` or `PBS_ENDPOINT` is set, the `target` parameter is ignored.

## Node metrics

According to the [api documentation](https://pbs.proxmox.com/docs/api-viewer/index.html#/nodes/{node}), we have to provide a node name (won't work with the node ip), but it seems to work with any name, so we just use "localhost" for the request. This setup is tested with one proxmox backup server host.

## Supported versions

We have tested the exporter with Proxmox Backup Server version **3.X** (see [Proxmox Backup Server Roadmap](https://pbs.proxmox.com/wiki/index.php/Roadmap)). If you have already tested the exporter with a newer version, or have encountered problems, please let us know.

## Release

Each release of the application includes Go-binary archives, checksums file, SBOMs and container images. 

The release workflow creates provenance for its builds using the [SLSA standard](https://slsa.dev), which conforms to the [Level 3 specification](https://slsa.dev/spec/v1.0/levels#build-l3). Each artifact can be verified using the `slsa-verifier` or `cosign` tool (see [Release verification](SECURITY.md#release-verification)).
