# 🔥 Oven Demo

This demo uses an oven as a concrete, real-world example to make the purpose of a PID controller more intuitive, comparing it with a simple on-off controller. The simulation exports Prometheus metrics, which are visualized in Grafana.

## Start Prometheus and Grafana

```bash
podman compose up
```

This starts Prometheus and Grafana with the provided configuration and dashboard provisioning. Grafana is available at http://localhost:3000/d/adp6sw7/pid-controller-oven-demo.

## Run the Simulation

In a separate terminal:

```bash
go run main.go
```

This runs both controllers against the same oven model and exposes metrics at http://localhost:2112/metrics.


### Grafana Dashboard

The Grafana dashboard visualizes the oven simulation and controller behavior using the exported Prometheus metrics:

http://localhost:3000/d/adp6sw7/pid-controller-oven-demo

It shows the full control loop, including oven temperature, controller error, control signal, PID gains, and the proportional, integral, and derivative term contributions. Multiple controller configurations run against the same oven model how different tuning choices affect stability and response.

## Stopping and Resetting

To fully reset Grafana state and provisioning:

```sh
podman compose down -v
```

This is useful when testing changes to dashboards or data source provisioning.
