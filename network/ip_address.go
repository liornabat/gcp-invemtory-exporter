package network

import (
	"context"
	"github.com/liornabat/gcp_inventory_exporter/config"
	"github.com/liornabat/gcp_inventory_exporter/pkg/logger"
	"github.com/liornabat/gcp_inventory_exporter/project"
	"google.golang.org/api/compute/v1"
	"strings"
	"sync"
)

var ipAddressHeader = []string{
	"Project",
	"Region/Zone",
	"Name",
	"Address",
	"Network",
	"Subnetwork",
	"Address Type",
	"Used By",
	"Creation Timestamp",
}

func GetIPAddressInventory(ctx context.Context, projectsId []*project.Project, zones config.Zones, log *logger.Logger) ([][]string, error) {
	log.Infof("Getting IP address inventory")
	defer log.Infof("Done getting IP address inventory")
	service, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}
	var inventory [][]string
	inventory = append(inventory, ipAddressHeader)
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(projectsId))
	for _, projectId := range projectsId {
		go func(projectId *project.Project) {
			defer wg.Done()
			var localInventory [][]string
			log.Infof("Getting Ip Address inventory for project %s", projectId.Name)
			for _, zone := range zones {
				log.Infof("Getting Ip Address inventory for compute instances in zone %s", zone)
				instances, err := service.Instances.List(projectId.ID, zone).Do()
				if err != nil {
					log.Errorf("Failed to get compute inventory for project %s and zone %s, error: %s", projectId.Name, zone, err.Error())
					continue
				}
				for _, instance := range instances.Items {
					for _, networkInterface := range instance.NetworkInterfaces {
						localInventory = append(localInventory, []string{
							projectId.Name,
							removeUrlPrefix(zone),
							networkInterface.Name,
							networkInterface.NetworkIP,
							removeUrlPrefix(networkInterface.Network),
							removeUrlPrefix(networkInterface.Subnetwork),
							"INTERNAL",
							instance.Name,
							instance.CreationTimestamp,
						})
					}
				}
			}
			log.Infof("Getting Ip Address inventory with Aggregated List")
			req := service.Addresses.AggregatedList(projectId.ID)
			if err := req.Pages(ctx, func(page *compute.AddressAggregatedList) error {
				for _, item := range page.Items {
					for _, address := range item.Addresses {
						localInventory = append(localInventory, []string{
							projectId.Name,
							removeUrlPrefix(address.Region),
							address.Name,
							address.Address,
							removeUrlPrefix(address.Network),
							removeUrlPrefix(address.Subnetwork),
							address.AddressType,
							strings.Join(removeUrlPrefixes(address.Users), ","),
							address.CreationTimestamp,
						})
					}
				}
				return nil
			}); err != nil {
				log.Errorf("Failed to get IP address inventory for project %s, error: %s", projectId.Name, err.Error())

			}
			log.Infof("Getting Ip Address inventory with Global List")
			newReq := service.GlobalAddresses.List(projectId.ID)
			if err := newReq.Pages(ctx, func(page *compute.AddressList) error {
				for _, address := range page.Items {
					localInventory = append(localInventory, []string{
						projectId.Name,
						"global",
						address.Name,
						address.Address,
						removeUrlPrefix(address.Network),
						removeUrlPrefix(address.Subnetwork),
						address.AddressType,
						strings.Join(removeUrlPrefixes(address.Users), ","),
						address.CreationTimestamp,
					})
				}
				return nil
			}); err != nil {
				log.Errorf("Failed to get IP address inventory for project %s, error: %s", projectId.Name, err.Error())
			}
			mutex.Lock()
			inventory = append(inventory, localInventory...)
			mutex.Unlock()
		}(projectId)
	}
	wg.Wait()

	return inventory, nil
}
