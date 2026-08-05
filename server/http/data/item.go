package data

type ItemVO struct {
	ID   int    `json:"id"`
	Name string `json:"name" binding:"required"`
}
