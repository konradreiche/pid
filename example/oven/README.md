# 🔥 Oven Demo

This demo uses an oven as a concrete, real-world example to make the purpose of a PID controller more intuitive, comparing it with a simple on-off controller. The simulation exports Prometheus metrics, which are visualized in Grafana.

## Start Prometheus and Grafana

```bash
podman compose up
```

This starts Prometheus and Grafana with the provided configuration and dashboard provisioning. Grafana is available at http://localhost:3000.

## Run the Simulation

In a separate terminal:

```bash
go run main.go
```

This runs both controllers against the same oven model and exposes metrics at http://localhost:2112/metrics.

## Stopping and Resetting

To fully reset Grafana state and provisioning:

```sh
podman compose down -v
```

This is useful when testing changes to dashboards or data source provisioning.
