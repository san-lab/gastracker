#!/bin/bash


#TODO get grafana docker id

GRAFANA_DOCKERID=$(docker ps | egrep -o "([0-9a-f]{12})(\s+grafana)" | cut -c1-12)

# install the histogram plugin
docker exec $GRAFANA_DOCKERID grafana-cli plugins install mtanda-histogram-panel && \
# setup the InfluxDB data source
curl 'http://admin:awesome_gastracker@localhost:3000/api/datasources' \
    -X POST \
    -H 'Content-Type: application/json; charset=UTF-8' \
    -d @grafana-influx.json && \
# import the dashboard
curl 'http://admin:awesome_gastracker@localhost:3000/api/dashboards/db' \
    -X POST \
    -H 'Content-Type: application/json; charset=UTF-8' \
    -d "{\"overwrite\":true,\"dashboard\": $(cat grafana-dashboard.json | sed 's/${DS_INFLUXDB}/InfluxDB/g')}" && \
# restart the container
docker restart $GRAFANA_DOCKERID