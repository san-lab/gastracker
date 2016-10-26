package main

import (
	"github.com/influxdata/influxdb/client/v2"
	"fmt"
	"time"
	"log"
	"errors"
	"strconv"
)

const (
	DB_NAME = "gastracker"
	DB_PRECISION = "s"
	DB_SERIES = "gastracking"
)

type Influx struct {
	client client.Client
}

// The interface with the Influx database.
func StartInflux() (*Influx, error) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     *influxEndpoint,
		Username: "gastracker",
		Password: "gastracker",
	})
	if err != nil {
		return nil, err
	}
	influx := &Influx{
		client: c,
	}
	if err := influx.createDatabase(); err != nil {
		return nil, err
	}
	return influx, nil
}

type TxPoint struct {
	Time     time.Time
	Token    string
	Gas      uint64
	GasPrice uint64
	FeeMap   map[string]float64
	Block    uint64
}

func (i *Influx) AddTxPoints(pnts []*TxPoint) error {
	bp, err := i.newBatchPoints()
	if err != nil {
		return err
	}
	for _, pnt := range pnts {
		tags := map[string]string{"token": pnt.Token}
		fields := make(map[string]interface{})
		if pnt.Token != "ETH" {
			fields["gas"] = pnt.Gas
			fields["block"] = pnt.Block
		}
		for k, v := range pnt.FeeMap {
			fields["fee_" + k] = v
		}
		pt, err := client.NewPoint(DB_SERIES, tags, fields, pnt.Time)
		if err != nil {
			return nil
		}
		bp.AddPoint(pt)
	}
	if err := i.client.Write(bp); err != nil {
		log.Printf("ERROR tring to write to InfluxDB: %s", err)
		return err
	}
	return nil
}

func (i *Influx) GetLatestPointBlock() (uint64, error) {
	q := client.NewQuery(fmt.Sprintf("SELECT last(block) FROM %s", DB_SERIES), DB_NAME, DB_PRECISION)
	resp, err := i.client.Query(q)
	if err != nil || resp.Error() != nil {
		return 0, err
	}
	if len(resp.Results) == 0 || resp.Results[0].Series == nil {
		return 0, nil
	}
	row := resp.Results[0].Series[0]
	for i, header := range row.Columns {
		if header == "last" {
			height, err := strconv.ParseUint(row.Values[0][i].(string), 10, 64)
			if err != nil {
				return 0, err
			}
			return height, nil
		}
	}
	return 0, errors.New("Could not find block height in ")
}

func (i *Influx) Stop() {
	log.Println("Influx is closing...")
	if err := i.client.Close(); err != nil {
		log.Printf("ERROR closing influx backend: %s\n", err)
	}
}

func (i *Influx) createDatabase() error {
	q := client.NewQuery(fmt.Sprintf("CREATE DATABASE %s", DB_NAME), "", "")
	if resp, err := i.client.Query(q); err != nil || resp.Error() != nil {
		return err
	}
	log.Println("Successfully created database")
	return nil
}

func (i *Influx) newBatchPoints() (client.BatchPoints, error) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: DB_NAME,
		Precision: DB_PRECISION,
	})
	if err != nil {
		return nil, err
	}
	return bp, nil
}
