package main

import (
	"encoding/json"
	"fmt"

	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
	"go.uber.org/zap"
)

// Unmarshalling events is complicated because of the fact that some redfish
// implementations do not return events based on the redfish standard.
func unmarshalEvents(bodyBytes []byte) (events hmcollector.Events, err error) {
	var jsonObj map[string]interface{}
	marshalErr := func(bb []byte, e error) {
		err = e
		logger.Error("Unable to unmarshal JSON payload",
			zap.ByteString("bodyString", bb),
			zap.Error(e),
		)
	}
	if e := json.Unmarshal(bodyBytes, &jsonObj); e != nil {
		marshalErr(bodyBytes, e)
		return
	}
	// We know we got some sort of JSON object.
	v, ok := jsonObj["Events"]
	if !ok {
		marshalErr(bodyBytes, fmt.Errorf("JSON payload missing Events"))
		return
	}
	delete(jsonObj, "Events")

	if newBodyBytes, e := json.Marshal(jsonObj); e != nil {
		marshalErr(bodyBytes, e)
		return
	} else if e = json.Unmarshal(newBodyBytes, &events); e != nil {
		marshalErr(newBodyBytes, e)
		return
	}

	// We now have the base events object, but without the Events array.
	// The variable v is holding this info right now. We need to process each
	// entry of this array individually, since the OriginOfCondition field may
	// be a string rather than a JSON object.

	if evBytes, e := json.Marshal(v); e != nil {
		marshalErr(bodyBytes, e)
		return
	} else {
		var evObjs []map[string]interface{}
		if e = json.Unmarshal(evBytes, &evObjs); e != nil {
			marshalErr(evBytes, e)
			return
		}
		for _, ev := range evObjs {
			s, ok := ev["OriginOfCondition"].(string)
			if ok {
				delete(ev, "OriginOfCondition")
			}
			tmp, e := json.Marshal(ev)
			var tmpEvent hmcollector.Event
			if e = json.Unmarshal(tmp, &tmpEvent); e != nil {
				marshalErr(tmp, e)
				return
			}
			if ok {
				// if OriginOfCondition was a string, we need
				// to put it into the unmarshalled event.
				tmpEvent.OriginOfCondition = &hmcollector.ResourceID{s}
			}
			events.Events = append(events.Events, tmpEvent)
		}
	}
	return
}
