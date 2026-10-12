package domain

type Counts struct {
	Tasks               int `json:"tasks"`
	Tickets             int `json:"tickets"`
	SubmittedIntakes    int `json:"submitted_intakes"`
	UnreadNotifications int `json:"unread_notifications"`
}
