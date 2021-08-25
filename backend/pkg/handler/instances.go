package handler

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

func (h *Handler) GetInstance(ctx echo.Context, appID string, groupID string, instanceID string) error {
	instance, err := h.db.GetInstance(instanceID, appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx.NoContent(http.StatusNotFound)
		}
		logger.Error().Err(err).Str("appID", appID).Str("instanceID", instanceID).Msg("getInstance - getting instance")
		return ctx.NoContent(http.StatusInternalServerError)
	}
	return ctx.JSON(http.StatusOK, instance)
}

func (h *Handler) GetInstanceStatusHistory(ctx echo.Context, appID string, groupID string, instanceID string, params codegen.GetInstanceStatusHistoryParams) error {
	instanceStatusHistory, err := h.db.GetInstanceStatusHistory(instanceID, appID, groupID, params.Limit)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx.NoContent(http.StatusNotFound)
		}
		logger.Error().Err(err).Str("appID", appID).Str("groupID", groupID).Str("instanceID", instanceID).Msgf("getInstanceStatusHistory - getting status history limit %d", params.Limit)
		return ctx.NoContent(http.StatusInternalServerError)
	}

	return ctx.JSON(http.StatusOK, instanceStatusHistory)
}

func (h *Handler) UpdateInstance(ctx echo.Context, instanceID string) error {
	logger := loggerWithUsername(logger, ctx)

	var request codegen.UpdateInstanceInfo

	err := ctx.Bind(&request)
	if err != nil {
		return ctx.NoContent(http.StatusBadRequest)
	}

	instance, err := h.db.UpdateInstance(instanceID, request.Alias)
	if err != nil {
		logger.Error().Err(err).Str("instance", instanceID).Msgf("updateInstance - updating params %s", request.Alias)
		return ctx.NoContent(http.StatusInternalServerError)
	}

	logger.Info().Msgf("updateInstance - successfully updated instance %q alias to %q", instanceID, instance.Alias)

	return ctx.JSON(http.StatusOK, instance)
}