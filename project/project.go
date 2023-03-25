package project

import (
	"context"
	"github.com/liornabat/gcp_inventory_exporter/pkg/logger"
	"google.golang.org/api/cloudresourcemanager/v1"
)

type Project struct {
	ID   string
	Name string
}

func GetProjects(ctx context.Context, log *logger.Logger) ([]*Project, error) {
	log.Infof("Getting projects list")
	defer log.Infof("Done getting projects list")
	service, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, err
	}

	req := service.Projects.List()
	var projects []*Project

	if err := req.Pages(ctx, func(page *cloudresourcemanager.ListProjectsResponse) error {
		for _, project := range page.Projects {
			log.Infof("Found project %s", project.Name)
			p := &Project{
				Name: project.Name,
				ID:   project.ProjectId,
			}
			projects = append(projects, p)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	log.Infof("Found %d projects", len(projects))
	return projects, nil
}
