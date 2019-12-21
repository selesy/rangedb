package jsonrecordiostream

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/inklabs/rangedb"
	"github.com/inklabs/rangedb/provider/jsonrecordserializer"
)

type jsonRecordIoStream struct {
	eventTypes map[string]reflect.Type
}

func New() *jsonRecordIoStream {
	return &jsonRecordIoStream{
		eventTypes: map[string]reflect.Type{},
	}
}

func (s *jsonRecordIoStream) Write(writer io.Writer, records <-chan *rangedb.Record) <-chan error {
	errors := make(chan error)
	go func() {
		totalSaved := 0

		_, _ = fmt.Fprint(writer, "[")

		for record := range records {
			data, err := json.Marshal(record)
			if err != nil {
				errors <- fmt.Errorf("failed marshalling event: %v", err)
			}

			if totalSaved > 0 {
				_, _ = fmt.Fprint(writer, ",")
			}

			_, _ = fmt.Fprintf(writer, "%s", data)

			totalSaved++
		}
		_, _ = fmt.Fprint(writer, "]")

		close(errors)
	}()

	return errors
}

func (s *jsonRecordIoStream) Read(reader io.Reader) (<-chan *rangedb.Record, <-chan error) {
	ch := make(chan *rangedb.Record)
	errors := make(chan error)

	go func() {
		defer close(errors)
		defer close(ch)

		decoder := json.NewDecoder(reader)
		decoder.UseNumber()

		_, err := decoder.Token()
		if err != nil {
			return
		}

		for decoder.More() {
			record, err := jsonrecordserializer.UnmarshalRecord(decoder, s.eventTypeLookup)
			if err != nil {
				errors <- err
				return
			}

			ch <- record
		}
	}()

	return ch, errors
}

func (s *jsonRecordIoStream) Bind(events ...rangedb.Event) {
	for _, e := range events {
		s.eventTypes[e.EventType()] = getType(e)
	}
}

func (s *jsonRecordIoStream) eventTypeLookup(eventTypeName string) (r reflect.Type, b bool) {
	eventType, ok := s.eventTypes[eventTypeName]
	return eventType, ok
}

func getType(object interface{}) reflect.Type {
	t := reflect.TypeOf(object)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t
}
