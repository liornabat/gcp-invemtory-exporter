package network

import (
	"context"
	"fmt"
	"github.com/liornabat/gcp_inventory_exporter/pkg/logger"
	"github.com/liornabat/gcp_inventory_exporter/project"
	"google.golang.org/api/compute/v1"
	"sync"
)

var peeringHeader = []string{
	"Project",
	"Name",
	"Network",
	"Peer Network",
	"State",
	"Auto Create Routes",
	"Exchange Subnet Routes",
	"Export Custom Routes",
	"Import Custom Routes",
	"Export Subnet Routes With Public IP",
	"Import Subnet Routes With Public IP",
	"Creation Timestamp",
}

func GetPreeingInventory(ctx context.Context, projectsId []*project.Project, log *logger.Logger) ([][]string, error) {
	log.Infof("Getting Peering inventory")
	defer log.Infof("Done Peering network inventory")
	service, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}
	var inventory [][]string
	inventory = append(inventory, peeringHeader)
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(projectsId))
	for _, projectId := range projectsId {
		go func(projectId *project.Project) {
			defer wg.Done()
			var localInventory [][]string
			log.Infof("Getting network peering inventory for project %s", projectId.Name)
			req := service.Networks.List(projectId.ID)
			if err := req.Pages(ctx, func(page *compute.NetworkList) error {
				for _, network := range page.Items {
					for _, peering := range network.Peerings {
						localInventory = append(localInventory, []string{
							projectId.Name,
							peering.Name,
							removeUrlPrefix(network.Name),
							removeUrlPrefix(peering.Network),
							peering.StateDetails,
							fmt.Sprintf("%t", peering.AutoCreateRoutes),
							fmt.Sprintf("%t", peering.ExchangeSubnetRoutes),
							fmt.Sprintf("%t", peering.ExportCustomRoutes),
							fmt.Sprintf("%t", peering.ImportCustomRoutes),
							fmt.Sprintf("%t", peering.ExportSubnetRoutesWithPublicIp),
							fmt.Sprintf("%t", peering.ImportSubnetRoutesWithPublicIp),
							network.CreationTimestamp,
						})
					}
				}
				return nil
			}); err != nil {
				log.Errorf("Failed to get network peering inventory for project %s , error: %s", projectId.Name, err.Error())
			}
			mutex.Lock()
			inventory = append(inventory, localInventory...)
			mutex.Unlock()
		}(projectId)
	}
	wg.Wait()
	return inventory, nil
}
