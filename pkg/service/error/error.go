package error

import (
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
)

func WriteResponse(c *app.RequestContext, err error, data interface{}) {
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{
			"Error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, map[string]any{
		"Result": data,
	})
}
