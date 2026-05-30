package subjects

import "fmt"

const Prefix = "constellation"

const AllEvents = Prefix + ".event.>"

func EventSubject(eventType string) string {
	return fmt.Sprintf("%s.event.%s", Prefix, eventType)
}

func EventSourceSubject(eventType, source string) string {
	return fmt.Sprintf("%s.event.%s.%s", Prefix, eventType, source)
}
