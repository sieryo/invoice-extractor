package job

import "time"

func NewJob(id string, userID *string, jobType JobType, payload []byte) *Job {
	now := time.Now()
	return &Job{
		ID:           id,
		UserID:       userID,
		Type:         jobType,
		Status:       JobPending,
		Progress:     0,
		InputPayload: payload,
		CreatedAt:    now,
	}
}
