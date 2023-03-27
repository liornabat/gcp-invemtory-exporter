package compute

import (
	"context"
	"fmt"
	"github.com/liornabat/gcp_inventory_exporter/config"
	"github.com/liornabat/gcp_inventory_exporter/pkg/logger"
	"github.com/liornabat/gcp_inventory_exporter/project"
	"google.golang.org/api/compute/v1"
	"strings"
	"sync"
)

var computeHeader = []string{
	"Project",
	"Zone",
	"Name",
	"Status",
	"Machine Type",
	"CPU",
	"Memory (MB)",
	"IP Address",
	"Disks (GB)",
	"Creation Time",
}

func getNetworkInterfaces(instance *compute.Instance) string {
	var networkInterfaces []string
	for _, networkInterface := range instance.NetworkInterfaces {
		networkInterfaces = append(networkInterfaces, networkInterface.NetworkIP)
	}
	return strings.Join(networkInterfaces, ", ")
}
func getDisksSizes(instance *compute.Instance) string {
	var disksSizes []string
	for _, disk := range instance.Disks {
		disksSizes = append(disksSizes, fmt.Sprintf("%dGB", disk.DiskSizeGb))
	}
	return strings.Join(disksSizes, ", ")
}
func removeUrlPrefix(url string) string {
	return strings.Split(url, "/")[len(strings.Split(url, "/"))-1]
}
func GetComputeInventory(ctx context.Context, projectsId []*project.Project, zones config.Zones, log *logger.Logger) ([][]string, error) {
	log.Infof("Getting compute inventory")
	defer log.Infof("Done getting compute inventory")
	service, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}
	var inventory [][]string
	inventory = append(inventory, computeHeader)
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(projectsId))
	for _, projectId := range projectsId {
		go func(projectId *project.Project) {
			defer wg.Done()
			var localInventory [][]string
			log.Infof("Getting compute inventory for project %s", projectId.Name)
			for _, zone := range zones {
				log.Infof("Getting compute inventory for compute instances in zone %s", zone)
				instances, err := service.Instances.List(projectId.ID, zone).Do()
				if err != nil {
					log.Errorf("Failed to get compute inventory for project %s and zone %s, error: %s", projectId.Name, zone, err.Error())
					continue
				}

				machineTypes := FetchMachineTypes(ctx, projectId.ID, zone, log)

				for _, instance := range instances.Items {
					mt := removeUrlPrefix(instance.MachineType)
					localInventory = append(localInventory, []string{
						projectId.Name,
						zone,
						instance.Name,
						instance.Status,
						mt,
						machineTypes.GetCPU(mt),
						machineTypes.GetMemory(mt),
						getNetworkInterfaces(instance),
						getDisksSizes(instance),
						instance.CreationTimestamp,
					})
				}
			}
			mutex.Lock()
			inventory = append(inventory, localInventory...)
			mutex.Unlock()
			log.Infof("Done getting compute inventory for project %s", projectId.Name)
		}(projectId)
	}
	wg.Wait()
	return inventory, nil
}
