/**
 * This is an example application that uses the helpers package.
 */
package main

import "github.com/gin-gonic/gin"

var app = gin.Default()

// Start server
func main() {
	app.Run(":3000")
}
