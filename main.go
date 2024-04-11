package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const promNamespace = "pbs"
const datastoreUsageApi = "/api2/json/status/datastore-usage"
const datastoreApi = "/api2/json/admin/datastore"
const nodeApi = "/api2/json/nodes"

var (
	timeoutDuration time.Duration

	tr = &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	client = &http.Client{
		Transport: tr,
	}

	// Flags
	endpoint = flag.String("pbs.endpoint", "",
		"Proxmox Backup Server endpoint")
	username = flag.String("pbs.username", "root@pam",
		"Proxmox Backup Server username")
	apitoken = flag.String("pbs.api.token", "",
		"Proxmox Backup Server API token")
	apitokenname = flag.String("pbs.api.token.name", "pbs-exporter",
		"Proxmox Backup Server API token name")
	timeout = flag.String("pbs.timeout", "5s",
		"Proxmox Backup Server timeout")
	insecure = flag.String("pbs.insecure", "false",
		"Proxmox Backup Server insecure")
	metricsPath = flag.String("pbs.metrics-path", "/metrics",
		"Path under which to expose metrics")
	listenAddress = flag.String("pbs.listen-address", ":9101",
		"Address on which to expose metrics")
	loglevel = flag.String("pbs.loglevel", "info",
		"Loglevel")

	// Metrics
	up = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "up"),
		"Was the last query of PBS successful.",
		nil, nil,
	)
	available = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "available"),
		"The available bytes of the underlying storage.",
		[]string{"datastore"}, nil,
	)
	size = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "size"),
		"The size of the underlying storage in bytes.",
		[]string{"datastore"}, nil,
	)
	used = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "used"),
		"The used bytes of the underlying storage.",
		[]string{"datastore"}, nil,
	)
	snapshot_count = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "snapshot_count"),
		"The total number of backups.",
		[]string{"datastore", "namespace"}, nil,
	)
	snapshot_vm_count = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "snapshot_vm_count"),
		"The total number of backups per VM.",
		[]string{"datastore", "namespace", "vm_id", "vm_name"}, nil,
	)
	snapshot_vm_last_timestamp = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "snapshot_vm_last_timestamp"),
		"The timestamp of the last backup of a VM.",
		[]string{"datastore", "namespace", "vm_id", "vm_name"}, nil,
	)
	snapshot_vm_last_verify = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "snapshot_vm_last_verify"),
		"The verify status of the last backup of a VM.",
		[]string{"datastore", "namespace", "vm_id", "vm_name"}, nil,
	)
	host_cpu_usage = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_cpu_usage"),
		"The CPU usage of the host.",
		nil, nil,
	)
	host_memory_free = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_memory_free"),
		"The free memory of the host.",
		nil, nil,
	)
	host_memory_total = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_memory_total"),
		"The total memory of the host.",
		nil, nil,
	)
	host_memory_used = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_memory_used"),
		"The used memory of the host.",
		nil, nil,
	)
	host_swap_free = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_swap_free"),
		"The free swap of the host.",
		nil, nil,
	)
	host_swap_total = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_swap_total"),
		"The total swap of the host.",
		nil, nil,
	)
	host_swap_used = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_swap_used"),
		"The used swap of the host.",
		nil, nil,
	)
	host_disk_available = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_disk_available"),
		"The available disk of the local root disk in bytes.",
		nil, nil,
	)
	host_disk_total = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_disk_total"),
		"The total disk of the local root disk in bytes.",
		nil, nil,
	)
	host_disk_used = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_disk_used"),
		"The used disk of the local root disk in bytes.",
		nil, nil,
	)
	host_uptime = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_uptime"),
		"The uptime of the host.",
		nil, nil,
	)
	host_io_wait = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_io_wait"),
		"The io wait of the host.",
		nil, nil,
	)
	host_load1 = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_load1"),
		"The load for 1 minute of the host.",
		nil, nil,
	)
	host_load5 = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_load5"),
		"The load for 5 minutes of the host.",
		nil, nil,
	)
	host_load15 = prometheus.NewDesc(
		prometheus.BuildFQName(promNamespace, "", "host_load15"),
		"The load for 15 minutes of the host.",
		nil, nil,
	)
)

type DatastoreResponse struct {
	Data []struct {
		Avail     int64  `json:"avail"`
		Store     string `json:"store"`
		Total     int64  `json:"total"`
		Used      int64  `json:"used"`
		Namespace string `json:"ns"`
	} `json:"data"`
}

type Datastore struct {
	Avail     int64  `json:"avail"`
	Store     string `json:"store"`
	Total     int64  `json:"total"`
	Used      int64  `json:"used"`
	Namespace string `json:"ns"`
}

type NamespaceResponse struct {
	Data []struct {
		Namespace string `json:"ns"`
	} `json:"data"`
}

type SnapshotResponse struct {
	Data []struct {
		BackupID     string `json:"backup-id"`
		BackupTime   int64  `json:"backup-time"`
		VMName       string `json:"comment"`
		Verification struct {
			State string `json:"state"`
		} `json:"verification"`
	} `json:"data"`
}

type HostResponse struct {
	Data struct {
		CPU float64 `json:"cpu"`
		Mem struct {
			Free  int64 `json:"free"`
			Total int64 `json:"total"`
			Used  int64 `json:"used"`
		} `json:"memory"`
		Swap struct {
			Free  int64 `json:"free"`
			Total int64 `json:"total"`
			Used  int64 `json:"used"`
		} `json:"swap"`
		Disk struct {
			Avail int64 `json:"avail"`
			Total int64 `json:"total"`
			Used  int64 `json:"used"`
		} `json:"root"`
		Load   []float64 `json:"loadavg"`
		Uptime int64     `json:"uptime"`
		Wait   float64   `json:"wait"`
	} `json:"data"`
}

type Exporter struct {
	endpoint            string
	authorizationHeader string
}

func ReadSecretFile(secretfilename string) string {
	file, err := os.Open(secretfilename)
	// flag to check the file format
	if err != nil {
		log.Fatal(err)
	}
	// Close the file
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	// Read the first line
	line := bufio.NewScanner(file)
	line.Scan()
	return line.Text()
}

func NewExporter(endpoint string, username string, apitoken string, apitokenname string) *Exporter {
	return &Exporter{
		endpoint:            endpoint,
		authorizationHeader: "PBSAPIToken=" + username + "!" + apitokenname + ":" + apitoken,
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- available
	ch <- size
	ch <- used
	ch <- snapshot_count
	ch <- snapshot_vm_count
	ch <- snapshot_vm_last_timestamp
	ch <- snapshot_vm_last_verify
	ch <- host_cpu_usage
	ch <- host_memory_free
	ch <- host_memory_total
	ch <- host_memory_used
	ch <- host_swap_free
	ch <- host_swap_total
	ch <- host_swap_used
	ch <- host_disk_available
	ch <- host_disk_total
	ch <- host_disk_used
	ch <- host_uptime
	ch <- host_io_wait
	ch <- host_load1
	ch <- host_load5
	ch <- host_load15
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	err := e.collectFromAPI(ch)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0,
		)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1,
	)

}

func (e *Exporter) collectFromAPI(ch chan<- prometheus.Metric) error {
	// get datastores
	req, err := http.NewRequest("GET", e.endpoint+datastoreUsageApi, nil)
	if err != nil {
		return err
	}

	// add Authorization header
	req.Header.Set("Authorization", e.authorizationHeader)

	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: Request URL: %s", req.URL)
		//log.Printf("DEBUG: Request Header: %s", vmID)
	}

	// make request and show output
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	// check if status code is 200
	if resp.StatusCode != 200 {
		return fmt.Errorf("ERROR: Status code %d returned from endpoint: %s", resp.StatusCode, e.endpoint)
	}

	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: Status code %d returned from endpoint: %s", resp.StatusCode, e.endpoint)
		//log.Printf("DEBUG: Response body: %s", string(body))
	}

	// parse json
	var response DatastoreResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	// for each datastore collect metrics
	for _, datastore := range response.Data {
		err := e.getDatastoreMetric(datastore, ch)
		if err != nil {
			return err
		}
	}

	// get node metrics
	err = e.getNodeMetrics(ch)
	if err != nil {
		return err
	}

	return nil
}

func (e *Exporter) getNodeMetrics(ch chan<- prometheus.Metric) error {
	// NOTE: According to the api documentation, we have to provide the node name (won't work with the node ip),
	// but it seems to work with any name, so we just use "localhost" here.
	// see: https://pbs.proxmox.com/docs/api-viewer/index.html#/nodes/{node}
	req, err := http.NewRequest("GET", e.endpoint+nodeApi+"/localhost/status", nil)
	if err != nil {
		return err
	}

	// add Authorization header
	req.Header.Set("Authorization", e.authorizationHeader)

	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: Request URL: %s", req.URL)
		//log.Printf("DEBUG: Request Header: %s", vmID)
	}

	// make request and show output
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	// check if status code is 200
	if resp.StatusCode != 200 {
		return fmt.Errorf("ERROR: Status code %d returned from endpoint: %s", resp.StatusCode, e.endpoint)
	}

	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: Status code %d returned from endpoint: %s", resp.StatusCode, e.endpoint)
		//log.Printf("DEBUG: Response body: %s", string(body))
	}

	// parse json
	var response HostResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	// set host metrics
	ch <- prometheus.MustNewConstMetric(
		host_cpu_usage, prometheus.GaugeValue, float64(response.Data.CPU),
	)
	ch <- prometheus.MustNewConstMetric(
		host_memory_free, prometheus.GaugeValue, float64(response.Data.Mem.Free),
	)
	ch <- prometheus.MustNewConstMetric(
		host_memory_total, prometheus.GaugeValue, float64(response.Data.Mem.Total),
	)
	ch <- prometheus.MustNewConstMetric(
		host_memory_used, prometheus.GaugeValue, float64(response.Data.Mem.Used),
	)
	ch <- prometheus.MustNewConstMetric(
		host_swap_free, prometheus.GaugeValue, float64(response.Data.Swap.Free),
	)
	ch <- prometheus.MustNewConstMetric(
		host_swap_total, prometheus.GaugeValue, float64(response.Data.Swap.Total),
	)
	ch <- prometheus.MustNewConstMetric(
		host_swap_used, prometheus.GaugeValue, float64(response.Data.Swap.Used),
	)
	ch <- prometheus.MustNewConstMetric(
		host_disk_available, prometheus.GaugeValue, float64(response.Data.Disk.Avail),
	)
	ch <- prometheus.MustNewConstMetric(
		host_disk_total, prometheus.GaugeValue, float64(response.Data.Disk.Total),
	)
	ch <- prometheus.MustNewConstMetric(
		host_disk_used, prometheus.GaugeValue, float64(response.Data.Disk.Used),
	)
	ch <- prometheus.MustNewConstMetric(
		host_uptime, prometheus.GaugeValue, float64(response.Data.Uptime),
	)
	ch <- prometheus.MustNewConstMetric(
		host_io_wait, prometheus.GaugeValue, float64(response.Data.Wait),
	)
	ch <- prometheus.MustNewConstMetric(
		host_load1, prometheus.GaugeValue, float64(response.Data.Load[0]),
	)
	ch <- prometheus.MustNewConstMetric(
		host_load5, prometheus.GaugeValue, float64(response.Data.Load[1]),
	)
	ch <- prometheus.MustNewConstMetric(
		host_load15, prometheus.GaugeValue, float64(response.Data.Load[2]),
	)

	return nil
}

func (e *Exporter) getDatastoreMetric(datastore Datastore, ch chan<- prometheus.Metric) error {
	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: --Store %s", datastore.Store)
		log.Printf("DEBUG: --Avail %d", datastore.Avail)
		log.Printf("DEBUG: --Total %d", datastore.Total)
		log.Printf("DEBUG: --Used %d", datastore.Used)
	}

	// set datastore metrics
	ch <- prometheus.MustNewConstMetric(
		available, prometheus.GaugeValue, float64(datastore.Avail), datastore.Store,
	)
	ch <- prometheus.MustNewConstMetric(
		size, prometheus.GaugeValue, float64(datastore.Total), datastore.Store,
	)
	ch <- prometheus.MustNewConstMetric(
		used, prometheus.GaugeValue, float64(datastore.Used), datastore.Store,
	)

	// get namespaces of datastore
	req, err := http.NewRequest("GET", e.endpoint+datastoreApi+"/"+datastore.Store+"/namespace", nil)
	if err != nil {
		return err
	}

	// add Authorization header
	req.Header.Set("Authorization", e.authorizationHeader)

	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: --Request URL: %s", req.URL)
		//log.Printf("DEBUG: --Request Header: %s", vmID)
	}

	// make request and show output
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	// check if status code is 200
	if resp.StatusCode != 200 {
		if resp.StatusCode == 400 {
			// check if datastore is being deleted
			isBeingDeleted, err := regexp.MatchString("(?i)datastore is being deleted", string(body[:]))
			if err != nil {
				return err
			}
			if isBeingDeleted {
				log.Printf("INFO: Datastore: %s is being deleted, Skip scrape datastore metric", datastore.Store)
				return nil
			}
		}
		return fmt.Errorf("ERROR: --Status code %d returned from endpoint: %s", resp.StatusCode, e.endpoint)
	}

	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: --Status code %d returned from endpoint: %s", resp.StatusCode, e.endpoint)
		//log.Printf("DEBUG: Response body: %s", string(body))
	}

	// parse json
	var response NamespaceResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	// for each namespace collect metrics
	for _, namespace := range response.Data {
		// if namespace is empty skip
		if namespace.Namespace == "" {
			continue
		}

		err := e.getNamespaceMetric(datastore.Store, namespace.Namespace, ch)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Exporter) getNamespaceMetric(datastore string, namespace string, ch chan<- prometheus.Metric) error {
	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: ----Namespace %s", namespace)
	}

	// get snapshots of datastore
	req, err := http.NewRequest("GET", e.endpoint+datastoreApi+"/"+datastore+"/snapshots?ns="+namespace, nil)
	if err != nil {
		return err
	}

	// add Authorization header
	req.Header.Set("Authorization", e.authorizationHeader)

	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: ----Request URL: %s", req.URL)
		//log.Printf("DEBUG: ----Request Header: %s", vmID)
	}

	// make request and show output
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	// check if status code is 200
	if resp.StatusCode != 200 {
		return fmt.Errorf("ERROR: ----Status code %d returned from endpoint: %s", resp.StatusCode, e.endpoint)
	}

	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: ----Status code %d returned from endpoint: %s", resp.StatusCode, e.endpoint)
		//log.Printf("DEBUG: Response body: %s", string(body))
	}

	// parse json
	var response SnapshotResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	// set total snapshot metrics
	ch <- prometheus.MustNewConstMetric(
		snapshot_count, prometheus.GaugeValue, float64(len(response.Data)), datastore, namespace,
	)

	// set snapshot metrics per vm
	vmNameMapping := make(map[string]string)
	vmCount := make(map[string]int)
	for _, snapshot := range response.Data {
		// get vm name from snapshot
		vmID := snapshot.BackupID
		vmNameMapping[vmID] = snapshot.VMName
		vmCount[vmID]++
	}

	// set snapshot metrics per vm
	for vmID, count := range vmCount {
		ch <- prometheus.MustNewConstMetric(
			snapshot_vm_count, prometheus.GaugeValue, float64(count), datastore, namespace, vmID, vmNameMapping[vmID],
		)

		// find last snapshot with backupID
		lastTimeStamp, lastVerify, err := findLastSnapshotWithBackupID(response, vmID)
		if err != nil {
			return err
		}
		lastVerifyBool := 0
		if lastVerify == "ok" {
			lastVerifyBool = 1
		}
		ch <- prometheus.MustNewConstMetric(
			snapshot_vm_last_timestamp, prometheus.GaugeValue, float64(lastTimeStamp), datastore, namespace, vmID, vmNameMapping[vmID],
		)
		ch <- prometheus.MustNewConstMetric(
			snapshot_vm_last_verify, prometheus.GaugeValue, float64(lastVerifyBool), datastore, namespace, vmID, vmNameMapping[vmID],
		)
	}

	return nil
}

func findLastSnapshotWithBackupID(response SnapshotResponse, backupID string) (int64, string, error) {
	// find biggest value of backupTime of backupID in response array
	var lastTimeStamp int64
	var lastVerify string
	for _, snapshot := range response.Data {
		if snapshot.BackupID == backupID {
			if snapshot.BackupTime > lastTimeStamp {
				lastTimeStamp = snapshot.BackupTime
				lastVerify = snapshot.Verification.State
			}
		}
	}

	// if lastTimeStamp is still 0, no snapshot was found
	if lastTimeStamp != 0 {
		return lastTimeStamp, lastVerify, nil
	}

	return 0, "", fmt.Errorf("ERROR: No snapshot found with backupID %s", backupID)
}

func main() {
	flag.Parse()

	// if env variable is set, it will overwrite defaults or flags
	if os.Getenv("PBS_LOGLEVEL") != "" {
		*loglevel = os.Getenv("PBS_LOGLEVEL")
	}
	if os.Getenv("PBS_ENDPOINT") != "" {
		*endpoint = os.Getenv("PBS_ENDPOINT")
	}
	if os.Getenv("PBS_USERNAME") != "" {
		*username = os.Getenv("PBS_USERNAME")
	} else {
		if os.Getenv("PBS_USERNAME_FILE") != "" {
			*username = ReadSecretFile(os.Getenv("PBS_USERNAME_FILE"))
		}
	}
	if os.Getenv("PBS_API_TOKEN_NAME") != "" {
		*apitokenname = os.Getenv("PBS_API_TOKEN_NAME")
	} else {
		if os.Getenv("PBS_API_TOKEN_NAME_FILE") != "" {
			*apitokenname = ReadSecretFile(os.Getenv("PBS_API_TOKEN_NAME_FILE"))
		}
	}
	if os.Getenv("PBS_API_TOKEN") != "" {
		*apitoken = os.Getenv("PBS_API_TOKEN")
	} else {
		if os.Getenv("PBS_API_TOKEN_FILE") != "" {
			*apitoken = ReadSecretFile(os.Getenv("PBS_API_TOKEN_FILE"))
		}
	}
	if os.Getenv("PBS_TIMEOUT") != "" {
		*timeout = os.Getenv("PBS_TIMEOUT")
	}
	if os.Getenv("PBS_INSECURE") != "" {
		*insecure = os.Getenv("PBS_INSECURE")
	}
	if os.Getenv("PBS_METRICS_PATH") != "" {
		*metricsPath = os.Getenv("PBS_METRICS_PATH")
	}
	if os.Getenv("PBS_LISTEN_ADDRESS") != "" {
		*listenAddress = os.Getenv("PBS_LISTEN_ADDRESS")
	}

	// convert flags
	insecureBool, err := strconv.ParseBool(*insecure)
	if err != nil {
		log.Fatalf("ERROR: Unable to parse insecure: %s", err)
	}

	// set insecure
	if insecureBool {
		tr.TLSClientConfig.InsecureSkipVerify = true
	}

	// set timeout
	timeoutDuration, err := time.ParseDuration(*timeout)
	if err != nil {
		log.Fatalf("ERROR: Unable to parse timeout: %s", err)
	}
	client.Timeout = timeoutDuration

	// debug
	if *loglevel == "debug" {
		log.Printf("DEBUG: Using connection endpoint: %s", *endpoint)
		log.Printf("DEBUG: Using connection username: %s", *username)
		log.Printf("DEBUG: Using connection apitoken: %s", *apitoken)
		log.Printf("DEBUG: Using connection apitokenname: %s", *apitokenname)
		log.Printf("DEBUG: Using connection timeout: %s", client.Timeout)
		log.Printf("DEBUG: Using connection insecure: %t", tr.TLSClientConfig.InsecureSkipVerify)
		log.Printf("DEBUG: Using metrics path: %s", *metricsPath)
		log.Printf("DEBUG: Using listen address: %s", *listenAddress)
	}

	if *endpoint != "" {
		log.Printf("INFO: Using fix connection endpoint: %s", *endpoint)
	}
	log.Printf("INFO: Listening on: %s", *listenAddress)
	log.Printf("INFO: Metrics path: %s", *metricsPath)

	// start http server
	http.HandleFunc(*metricsPath, func(w http.ResponseWriter, r *http.Request) {
		target := ""

		// if endpoint was not set as flag or env variable, we try to get it from "target" query parameter
		if *endpoint != "" {
			target = *endpoint
		} else {
			target = r.URL.Query().Get("target")
			if target == "" {
				// if target is not set, we use the default
				target = "http://localhost:8007"
			}
		}

		// debug
		if *loglevel == "debug" {
			log.Printf("DEBUG: Using connection endpoint %s", target)
		}

		exporter := NewExporter(target, *username, *apitoken, *apitokenname)

		// catch if register of exporter fails
		err := prometheus.Register(exporter)
		if err != nil {
			// if register fails, we log the error and return
			log.Printf("ERROR: %s", err)
		}
		promhttp.Handler().ServeHTTP(w, r) // Serve the metrics
		prometheus.Unregister(exporter)    // Clean up after serving
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>PBS Exporter</title></head>
            <body>
            <h1>Proxmox Backup Server Exporter</h1>
            <p><a href='` + *metricsPath + `'>Metrics</a></p>
            </body>
            </html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
