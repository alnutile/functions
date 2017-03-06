package server

import (
	"context"
	"net/http"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/iron-io/functions/api"
	"github.com/iron-io/functions/api/models"
	"github.com/iron-io/functions/api/runner/task"
	"github.com/iron-io/runner/common"
)

func (s *Server) handleRouteUpdate(c *gin.Context) {
	ctx := c.MustGet("ctx").(context.Context)
	log := common.Logger(ctx)

	var wroute models.RouteWrapper

	err := c.BindJSON(&wroute)
	if err != nil {
		log.WithError(err).Debug(models.ErrInvalidJSON)
		c.JSON(http.StatusBadRequest, simpleError(models.ErrInvalidJSON))
		return
	}

	if wroute.Route == nil {
		log.Debug(models.ErrRoutesMissingNew)
		c.JSON(http.StatusBadRequest, simpleError(models.ErrRoutesMissingNew))
		return
	}

	if wroute.Route.Path != "" {
		log.Debug(models.ErrRoutesPathImmutable)
		c.JSON(http.StatusBadRequest, simpleError(models.ErrRoutesPathImmutable))
		return
	}

	wroute.Route.AppName = c.MustGet(api.AppName).(string)
	wroute.Route.Path = path.Clean(c.MustGet(api.Path).(string))

	if err := wroute.Validate(true); err != nil {
		log.WithError(err).Debug(models.ErrRoutesUpdate)
		c.JSON(http.StatusBadRequest, simpleError(err))
		return
	}

	if wroute.Route.Image != "" {
		authCfg, err := s.DockerAuth.GetAuthConfiguration(ctx)
		if err != nil {
			log.WithError(err).Error(models.ErrInvalidDockerCreds)
			c.JSON(http.StatusInternalServerError, simpleError(models.ErrInvalidDockerCreds))
			return
		}

		err = s.Runner.EnsureImageExists(ctx, &task.Config{
			Image:             wroute.Route.Image,
			AuthConfiguration: *authCfg,
		})

		if err != nil {
			log.WithError(err).Debug(models.ErrRoutesUpdate)
			c.JSON(http.StatusBadRequest, simpleError(models.ErrUsableImage))
			return
		}
	}

	route, err := s.Datastore.UpdateRoute(ctx, wroute.Route)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	s.cacherefresh(route)

	c.JSON(http.StatusOK, routeResponse{"Route successfully updated", route})
}
