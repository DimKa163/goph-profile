package rest

import (
	"github.com/labstack/echo/v5"
)

// Section identifies a web page section.
type Section interface {
	// GET is the HTTP GET method section.
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) echo.RouteInfo
	// POST is the HTTP POST method section.
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) echo.RouteInfo
	// PUT is the HTTP PUT method section.
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) echo.RouteInfo
	// DELETE is the HTTP DELETE method section.
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) echo.RouteInfo
	// PATCH is the HTTP PATCH method section.
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) echo.RouteInfo
}
