package network

import (
	"context"
	"fmt"
	"github.com/liornabat/gcp_inventory_exporter/pkg/logger"
	"github.com/liornabat/gcp_inventory_exporter/project"
	"google.golang.org/api/compute/v1"
	"sync"
)

var routesHeader = []string{
	"Project",
	"Name",
	"Network",
	"Dest Range",
	"Priority",
	"Next Hop IP",
	"Next Hop Network",
	"Next Hop Gateway",
	"Next Hop Peering",
	"Next Hop Ilb",
	"Creation Timestamp",
}

func GetRoutesInventory(ctx context.Context, projectsId []*project.Project, log *logger.Logger) ([][]string, error) {
	log.Infof("Getting Routing inventory")
	defer log.Infof("Done Routing network inventory")
	service, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}
	var inventory [][]string
	inventory = append(inventory, routesHeader)
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(projectsId))
	for _, projectId := range projectsId {
		go func(projectId *project.Project) {
			defer wg.Done()
			var localInventory [][]string
			log.Infof("Getting routes inventory for project %s", projectId.Name)
			req := service.Routes.List(projectId.ID)
			if err := req.Pages(ctx, func(page *compute.RouteList) error {
				for _, route := range page.Items {
					localInventory = append(localInventory, []string{
						projectId.Name,
						route.Name,
						removeUrlPrefix(route.Network),
						route.DestRange,
						fmt.Sprintf("%d", route.Priority),
						removeUrlPrefix(route.NextHopIp),
						removeUrlPrefix(route.NextHopNetwork),
						removeUrlPrefix(route.NextHopGateway),
						removeUrlPrefix(route.NextHopPeering),
						removeUrlPrefix(route.NextHopIlb),
						route.CreationTimestamp,
					})
				}
				return nil
			}); err != nil {
				log.Errorf("Failed to get routes inventory for project %s , error: %s", projectId.Name, err.Error())
			}
			mutex.Lock()
			inventory = append(inventory, localInventory...)
			mutex.Unlock()
		}(projectId)
	}
	wg.Wait()
	return inventory, nil
}
