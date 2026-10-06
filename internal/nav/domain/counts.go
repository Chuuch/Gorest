package domain

type Counts struct {
	Tasks               int `json:"tasks"`
	Tickets             int `json:'tickets"`
	UnreadNotifications int `json:"unread_notifications"`
}
