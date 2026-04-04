package queue

import "github.com/gauravprasad/clawcontrol/internal/domain"

type Handler interface {
	Handle(job domain.Job) error
}
