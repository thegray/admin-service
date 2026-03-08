package rest

import "time"

type createExampleRequest struct {
	Param string `json:"param" binding:"required"`
}

type getExampleRequest struct {
	ID int `uri:"id" binding:"required,gt=0"`
}

type exampleResponse struct {
	ID        int64     `json:"id"`
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}
