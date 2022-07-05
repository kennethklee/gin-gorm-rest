package main

import "github.com/gin-gonic/gin"

/**
 * This is an example application that uses the helpers package.
 */

var app = gin.Default()

// Start server
func main() {
	app.Run(":3000")
}
