package app

import "testing-project/controllers"

func routes() {
	router.POST("/messages", controllers.CreateMessage)
	router.PUT("/messages/:message_id", controllers.UpdateMessage)
	router.DELETE("/messages/:message_id", controllers.DeleteMessage)
}
