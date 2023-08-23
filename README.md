# Proxmox Backup Server Exporter

<p align="center">
    <a href="https://github.com/natrontech/pbs-exporter"><img
    src="https://img.shields.io/github/license/natrontech/pbs-exporter"
    alt="License"
    /></a>
    <img alt="GitHub go.mod Go version" src="https://img.shields.io/github/go-mod/go-version/natrontech/pbs-exporter/main?label=Go%20Version" />
    <img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/natrontech/pbs-exporter/ci.yml?label=CI" />
    <img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/natrontech/pbs-exporter/codeql.yml?label=CodeQL" />
    <img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/natrontech/pbs-exporter/docker-release.yml?label=Docker%20Release" />
</p>

---

Export [Proxmox Backup Server](https://www.proxmox.com/en/proxmox-backup-server/overview) statistics to [Prometheus](https://prometheus.io/).

Metrics are retrieved using the [Proxmox Backup Server API](https://pbs.proxmox.com/docs/api-viewer/index.html).

## Exported Metrics

| Metric | Meaning | Labels |
| ------ | ------- | ------ |
| pbs_up | Was the last query of Proxmox Backup Server successful? | |
| pbs_available | The available bytes of the underlying storage. | `datastore` |
| pbs_size | The size of the underlying storage in bytes. | `datastore` |
| pbs_used | The used bytes of the underlying storage. | `datastore` |
| pbs_snapshot_count | The total number of backups. | `namespace` |
| pbs_snapshot_vm_count | The total number of backups per VM. | `namespace`, `vm_id` |
| pbs_host_cpu_usage | The CPU usage of the host. | |
| pbs_host_memory_free | The free memory of the host. | |
| pbs_host_memory_total | The total memory of the host. | |
| pbs_host_memory_used | The used memory of the host. | |
| pbs_host_swap_free | The free swap of the host. | |
| pbs_host_swap_total | The total swap of the host. | |
| pbs_host_swap_used | The used swap of the host. | |
| pbs_host_available_free | The available disk of the local root disk in bytes. | |
| pbs_host_disk_total | The total disk of the local root disk in bytes. | |
| pbs_host_disk_used | The used disk of the local root disk in bytes. | |
| pbs_host_uptime | The uptime of the host. | |
| pbs_host_io_wait | The io wait of the host. | |
| pbs_host_load1 | The load for 1 minute of the host. | |
| pbs_host_load5 | The load for 5 minutes of the host. | |
| pbs_host_load15 | The load 15 minutes of the host. | |

## Flags / Environment Variables

```bash
$ ./pbs-exporter -help
```

You can use the following flags to configure the exporter. All flags can also be set using environment variables. Environment variables take precedence over flags.

| Flag | Environment Variable | Description | Default |
| ---- | -------------------- | ----------- | ------- |
| `pbs.loglevl` | `PBS_LOGLEVEL` | Log level (debug, info) | `info` |
| `pbs.api.token` | `PBS_API_TOKEN` | API token to use for authentication | |
| `pbs.api.token.name` | `PBS_API_TOKEN_NAME` | Name of the API token to use for authentication | `pbs-exporter` |
| `pbs.endpoint` | `PBS_ENDPOINT` | Address of the Proxmox Backup Server | `http://localhost:8007` |
| `pbs.username` | `PBS_USERNAME` | Username to use for authentication | `root@pam` |
| `pbs.timeout` | `PBS_TIMEOUT` | Timeout for requests to Proxmox Backup Server | `5s` |
| `pbs.insecure` | `PBS_INSECURE` | Disable TLS certificate verification | `false` |
| `pbs.metrics-path` | `PBS_METRICS_PATH` | Path under which to expose metrics | `/metrics` |
| `pbs.web.listen-address` | `PBS_LISTEN_ADDRESS` | Address to listen on for web interface and telemetry | `:9101` |

## Node metrics

According to the [api documentation](https://pbs.proxmox.com/docs/api-viewer/index.html#/nodes/{node}), we have to provide a node name (won't work with the node ip), but it seems to work with any name, so we just use "localhost" for the request. This setup is tested with one proxmox backup server host.