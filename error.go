package apns

import (
	"fmt"
)

type NotificationError struct {
	Command      uint8
	Status       uint8
	Identifier   uint32
	Notification *Notification

	OtherError error
}

func NewNotificationError(p []byte, err error) (e *NotificationError) {
	e = new(NotificationError)
	if len(p) != 1+1+4 {
		if err != nil {
			e.OtherError = err
			return
		}
		e.OtherError = fmt.Errorf("Wrong data format, [%x]", p)
		return
	}
	e.Command = uint8(p[0])
	e.Status = uint8(p[1])
	e.Identifier = uint32(p[2])<<24 + uint32(p[3])<<16 + uint32(p[4])<<8 + uint32(p[5])
	return
}

func (e NotificationError) Error() string {
	if e.OtherError != nil {
		return e.OtherError.Error()
	}
	if e.Command != 8 {
		return fmt.Sprintf("Unknow error, command(%d), status(%d), id(%x)", e.Command, e.Status, e.Identifier)
	}
	status := ""
	switch e.Status {
	case 0:
		status = "No errors encountered"
	case 1:
		status = "Processing error"
	case 2:
		status = "Missing device token"
	case 3:
		status = "Missing topic"
	case 4:
		status = "Missing payload"
	case 5:
		status = "Invalid token size"
	case 6:
		status = "Invalid topic size"
	case 7:
		status = "Invalid payload size"
	case 8:
		status = "Invalid token"
	default:
		status = "None (unknown)"
	}
	return fmt.Sprintf("%s: id(%x),token(%s)", status, e.Identifier, e.Notification.DeviceToken)
}

func (e NotificationError) String() string {
	return e.Error()
}
