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

var networkHeader = []string{
	"Project",
	"Region",
	"Name",
	"Subnetwork",
	"CIDR",
	"Gateway Address",
	"Creation Timestamp",
}

func removeUrlPrefix(url string) string {
	return strings.Split(url, "/")[len(strings.Split(url, "/"))-1]
}
func removeUrlPrefixes(urls []string) []string {
	var newUrls []string
	for _, url := range urls {
		newUrls = append(newUrls, removeUrlPrefix(url))
	}
	return newUrls
}
func GetVPCInventory(ctx context.Context, projectsId []*project.Project, regions config.Regions, log *logger.Logger) ([][]string, error) {
	log.Infof("Getting network inventory")
	defer log.Infof("Done getting network inventory")
	service, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}
	var inventory [][]string
	inventory = append(inventory, networkHeader)
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(projectsId))
	for _, projectId := range projectsId {
		go func(projectId *project.Project) {
			defer wg.Done()
			var localInventory [][]string
			log.Infof("Getting network inventory for project %s", projectId.Name)
			for _, region := range regions {
				log.Infof("Getting network inventory for project %s in region %s", projectId.Name, region)
				req := service.Subnetworks.List(projectId.ID, region)
				if err := req.Pages(ctx, func(page *compute.SubnetworkList) error {
					for _, subnetwork := range page.Items {
						localInventory = append(localInventory, []string{
							projectId.Name,
							region,
							removeUrlPrefix(subnetwork.Network),
							subnetwork.Name,
							subnetwork.IpCidrRange,
							subnetwork.GatewayAddress,
							subnetwork.CreationTimestamp,
						})
					}
					return nil
				}); err != nil {
					log.Errorf("Failed to get subnetwork inventory for project %s in region %s, error: %s", projectId.Name, region, err.Error())
					continue
				}
			}
			mutex.Lock()
			inventory = append(inventory, localInventory...)
			mutex.Unlock()
		}(projectId)
	}
	wg.Wait()
	return inventory, nil
}
