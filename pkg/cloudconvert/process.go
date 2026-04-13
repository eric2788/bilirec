package cloudconvert

func NewVideoConvertTask(name string, payload *VideoConvertPayload) *JobCreateTask {
	if payload.AudioCodec == "" {
		payload.AudioCodec = "copy"
	}

	if payload.VideoCodec == "" {
		payload.VideoCodec = "copy"
	}

	return &JobCreateTask{
		Name:      name,
		Operation: "convert",
		Payload:   payload,
	}
}

func NewCommandTask(name string, payload *CommandPayload) *JobCreateTask {
	return &JobCreateTask{
		Name:      name,
		Operation: "command",
		Payload:   payload,
	}
}
