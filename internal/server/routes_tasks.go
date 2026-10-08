package server

import "net/http"

func (a *wiredApp) registerTasks(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/projects/{id}/tasks", a.staff(a.taskHandler.List))
	mux.Handle("GET /api/v1/inbox/tasks", a.staff(a.taskHandler.Inbox))
	mux.Handle("POST /api/v1/projects/{id}/tasks", a.staffIdempotent(a.taskHandler.Create))
	mux.Handle("PATCH /api/v1/tasks/{id}", a.staff(a.taskHandler.Update))
	mux.Handle("POST /api/v1/tickets/{id}/convert", a.staffIdempotent(a.taskHandler.Convert))
	mux.Handle("DELETE /api/v1/tasks/{id}", a.staff(a.taskHandler.Delete))

	mux.Handle("GET /api/v1/time-entries", a.staff(a.timeEntryHandler.ListRange))
	mux.Handle("GET /api/v1/tasks/{id}/time-entries", a.staff(a.timeEntryHandler.List))
	mux.Handle("POST /api/v1/tasks/{id}/time-entries", a.staff(a.timeEntryHandler.Create))
	mux.Handle("PATCH /api/v1/time-entries/{id}", a.staff(a.timeEntryHandler.Update))
	mux.Handle("DELETE /api/v1/time-entries/{id}", a.staff(a.timeEntryHandler.Delete))

	mux.Handle("GET /api/v1/projects/{id}/files", a.staff(a.fileHandler.List))
	mux.Handle("POST /api/v1/projects/{id}/files", a.staffIdempotent(a.fileHandler.Create))
	mux.Handle("DELETE /api/v1/files/{id}", a.staff(a.fileHandler.Delete))

	mux.Handle("GET /api/v1/tasks/{id}/comments", a.staff(a.commentHandler.List))
	mux.Handle("POST /api/v1/tasks/{id}/comments", a.staff(a.commentHandler.Create))
	mux.Handle("PATCH /api/v1/comments/{id}", a.staff(a.commentHandler.Update))
	mux.Handle("DELETE /api/v1/comments/{id}", a.staff(a.commentHandler.Delete))
}
