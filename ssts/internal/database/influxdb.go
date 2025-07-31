package database

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"

	"github.com/pranavgopavaram/ssts/internal/config"
	"github.com/pranavgopavaram/ssts/pkg/models"
)

// InfluxDB wraps InfluxDB client for time-series data
type InfluxDB struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
	queryAPI api.QueryAPI
	org      string
	bucket   string
}

// NewInfluxDB creates a new InfluxDB client
func NewInfluxDB(cfg config.InfluxDBConfig) *InfluxDB {
	client := influxdb2.NewClient(cfg.URL, cfg.Token)
	
	writeAPI := client.WriteAPI(cfg.Org, cfg.Bucket)
	queryAPI := client.QueryAPI(cfg.Org)

	// Setup error handling for write API
	go func() {
		for err := range writeAPI.Errors() {
			fmt.Printf("InfluxDB write error: %v\n", err)
		}
	}()

	return &InfluxDB{
		client:   client,
		writeAPI: writeAPI,
		queryAPI: queryAPI,
		org:      cfg.Org,
		bucket:   cfg.Bucket,
	}
}

// WriteMetricPoint writes a metric point to InfluxDB
func (idb *InfluxDB) WriteMetricPoint(point models.MetricPoint) error {
	p := influxdb2.NewPointWithMeasurement(point.Type).
		SetTime(point.Timestamp)

	// Add tags
	for k, v := range point.Tags {
		p = p.AddTag(k, v)
	}

	// Add test_id and source as tags
	p = p.AddTag("test_id", point.TestID).
		AddTag("source", point.Source)

	// Add fields
	for k, v := range point.Fields {
		p = p.AddField(k, v)
	}

	idb.writeAPI.WritePoint(p)
	return nil
}

// WriteSystemMetrics writes system metrics to InfluxDB
func (idb *InfluxDB) WriteSystemMetrics(testID string, metrics models.SystemMetrics) error {
	timestamp := metrics.Timestamp

	// CPU metrics
	cpuPoint := influxdb2.NewPointWithMeasurement("system_cpu").
		SetTime(timestamp).
		AddTag("test_id", testID).
		AddTag("host_id", "localhost"). // TODO: Get actual host ID
		AddField("usage_percent", metrics.CPU.UsagePercent).
		AddField("user_percent", metrics.CPU.UserPercent).
		AddField("system_percent", metrics.CPU.SystemPercent).
		AddField("idle_percent", metrics.CPU.IdlePercent).
		AddField("iowait_percent", metrics.CPU.IOWaitPercent).
		AddField("frequency_mhz", metrics.CPU.FrequencyMHz).
		AddField("temperature_celsius", metrics.CPU.Temperature)

	// Memory metrics
	memoryPoint := influxdb2.NewPointWithMeasurement("system_memory").
		SetTime(timestamp).
		AddTag("test_id", testID).
		AddTag("host_id", "localhost").
		AddTag("memory_type", "RAM").
		AddField("total_bytes", metrics.Memory.TotalBytes).
		AddField("used_bytes", metrics.Memory.UsedBytes).
		AddField("available_bytes", metrics.Memory.AvailableBytes).
		AddField("usage_percent", metrics.Memory.UsagePercent).
		AddField("swap_used_bytes", metrics.Memory.SwapUsedBytes).
		AddField("cache_bytes", metrics.Memory.CacheBytes).
		AddField("buffer_bytes", metrics.Memory.BufferBytes)

	// Disk metrics
	diskPoint := influxdb2.NewPointWithMeasurement("system_io").
		SetTime(timestamp).
		AddTag("test_id", testID).
		AddTag("host_id", "localhost").
		AddTag("device_name", "all").
		AddField("read_bytes_per_sec", metrics.Disk.ReadBytesPerSec).
		AddField("write_bytes_per_sec", metrics.Disk.WriteBytesPerSec).
		AddField("read_ops_per_sec", metrics.Disk.ReadOpsPerSec).
		AddField("write_ops_per_sec", metrics.Disk.WriteOpsPerSec).
		AddField("io_wait_percent", metrics.Disk.IOWaitPercent).
		AddField("queue_depth", metrics.Disk.QueueDepth).
		AddField("latency_ms", metrics.Disk.LatencyMs).
		AddField("usage_percent", metrics.Disk.UsagePercent)

	// Network metrics
	networkPoint := influxdb2.NewPointWithMeasurement("system_network").
		SetTime(timestamp).
		AddTag("test_id", testID).
		AddTag("host_id", "localhost").
		AddTag("interface_name", "all").
		AddField("rx_bytes_per_sec", metrics.Network.RxBytesPerSec).
		AddField("tx_bytes_per_sec", metrics.Network.TxBytesPerSec).
		AddField("rx_packets_per_sec", metrics.Network.RxPacketsPerSec).
		AddField("tx_packets_per_sec", metrics.Network.TxPacketsPerSec).
		AddField("rx_errors", metrics.Network.RxErrors).
		AddField("tx_errors", metrics.Network.TxErrors).
		AddField("latency_ms", metrics.Network.LatencyMs)

	// Write all points
	idb.writeAPI.WritePoint(cpuPoint)
	idb.writeAPI.WritePoint(memoryPoint)
	idb.writeAPI.WritePoint(diskPoint)
	idb.writeAPI.WritePoint(networkPoint)

	return nil
}

// WriteCustomMetrics writes plugin-specific metrics to InfluxDB
func (idb *InfluxDB) WriteCustomMetrics(testID, pluginName string, metrics map[string]interface{}) error {
	point := influxdb2.NewPointWithMeasurement("custom_metrics").
		SetTime(time.Now()).
		AddTag("test_id", testID).
		AddTag("plugin_name", pluginName)

	for k, v := range metrics {
		point = point.AddField(k, v)
	}

	idb.writeAPI.WritePoint(point)
	return nil
}

// QueryMetrics queries metrics from InfluxDB
func (idb *InfluxDB) QueryMetrics(ctx context.Context, testID string, measurement string, timeRange models.TimeRange) ([]models.MetricPoint, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "%s")
		|> filter(fn: (r) => r.test_id == "%s")
	`, idb.bucket, timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339), measurement, testID)

	result, err := idb.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer result.Close()

	var metrics []models.MetricPoint
	for result.Next() {
		record := result.Record()
		
		metric := models.MetricPoint{
			Timestamp: record.Time(),
			TestID:    testID,
			Source:    record.ValueByKey("source").(string),
			Type:      measurement,
			Tags:      make(map[string]string),
			Fields:    make(map[string]interface{}),
		}

		// Extract tags
		for k, v := range record.Values() {
			if k != "_time" && k != "_value" && k != "_field" && k != "_measurement" {
				if str, ok := v.(string); ok {
					metric.Tags[k] = str
				}
			}
		}

		// Extract field value
		fieldName := record.Field()
		fieldValue := record.Value()
		metric.Fields[fieldName] = fieldValue

		metrics = append(metrics, metric)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query result error: %w", result.Err())
	}

	return metrics, nil
}

// QuerySystemMetrics queries system metrics for a specific time range
func (idb *InfluxDB) QuerySystemMetrics(ctx context.Context, testID string, timeRange models.TimeRange) ([]models.SystemMetrics, error) {
	query := fmt.Sprintf(`
		import "join"
		
		cpu = from(bucket: "%s")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "system_cpu")
			|> filter(fn: (r) => r.test_id == "%s")
			|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
		
		memory = from(bucket: "%s")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "system_memory")
			|> filter(fn: (r) => r.test_id == "%s")
			|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
		
		disk = from(bucket: "%s")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "system_io")
			|> filter(fn: (r) => r.test_id == "%s")
			|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
		
		network = from(bucket: "%s")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "system_network")
			|> filter(fn: (r) => r.test_id == "%s")
			|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
		
		join.time(left: cpu, right: memory, fn: (l, r) => ({l with memory: r}))
	`, idb.bucket, timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339), testID,
		idb.bucket, timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339), testID,
		idb.bucket, timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339), testID,
		idb.bucket, timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339), testID)

	result, err := idb.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute system metrics query: %w", err)
	}
	defer result.Close()

	var systemMetrics []models.SystemMetrics
	for result.Next() {
		record := result.Record()
		// TODO: Parse the joined result into SystemMetrics struct
		// This is a simplified version - in practice, you'd need to handle the complex join result
		
		metric := models.SystemMetrics{
			Timestamp: record.Time(),
			// Parse CPU, Memory, Disk, Network from the record values
		}
		
		systemMetrics = append(systemMetrics, metric)
	}

	return systemMetrics, nil
}

// QueryLatestMetrics queries the latest metrics for a test
func (idb *InfluxDB) QueryLatestMetrics(ctx context.Context, testID string, measurement string, limit int) ([]models.MetricPoint, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: -1h)
		|> filter(fn: (r) => r._measurement == "%s")
		|> filter(fn: (r) => r.test_id == "%s")
		|> last()
		|> limit(n: %d)
	`, idb.bucket, measurement, testID, limit)

	result, err := idb.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute latest metrics query: %w", err)
	}
	defer result.Close()

	var metrics []models.MetricPoint
	for result.Next() {
		record := result.Record()
		
		metric := models.MetricPoint{
			Timestamp: record.Time(),
			TestID:    testID,
			Type:      measurement,
			Tags:      make(map[string]string),
			Fields:    make(map[string]interface{}),
		}

		// Extract tags and fields
		for k, v := range record.Values() {
			if k != "_time" && k != "_value" && k != "_field" && k != "_measurement" {
				if str, ok := v.(string); ok {
					metric.Tags[k] = str
				}
			}
		}

		fieldName := record.Field()
		fieldValue := record.Value()
		metric.Fields[fieldName] = fieldValue

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// CreateRetentionPolicies creates retention policies for data lifecycle management
func (idb *InfluxDB) CreateRetentionPolicies(ctx context.Context) error {
	// Note: InfluxDB 2.0 uses retention policies through the API
	// This would typically be configured through the InfluxDB UI or CLI
	// For demonstration, we'll skip the actual implementation
	return nil
}

// Flush forces any pending writes to be sent
func (idb *InfluxDB) Flush() {
	idb.writeAPI.Flush()
}

// Close closes the InfluxDB client
func (idb *InfluxDB) Close() {
	idb.writeAPI.Flush()
	idb.client.Close()
}

// HealthCheck performs a health check on InfluxDB
func (idb *InfluxDB) HealthCheck(ctx context.Context) error {
	health, err := idb.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("InfluxDB health check failed: %w", err)
	}

	if health.Status != "pass" {
		return fmt.Errorf("InfluxDB status: %s", health.Status)
	}

	return nil
}