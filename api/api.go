package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type Api struct{}

func (Api) ScheduleJob(ctx echo.Context) error {
	//TODO implement me
	return ctx.JSON(http.StatusOK, Job{
		Id:    0,
		RunAt: "@every 2 hours",
	})
}
