package response

import (
	"encoding/json"
	"net/http"
	"reflect"
)

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}

	payload = normalizePayload(payload)
	_ = json.NewEncoder(w).Encode(payload)
}

func normalizePayload(payload any) any {
	v := reflect.ValueOf(payload)
	switch v.Kind() {
	case reflect.Slice:
		if v.IsNil() {
			return reflect.MakeSlice(v.Type(), 0, 0).Interface()
		}
	case reflect.Struct:
		return normalizeStructSlices(v)
	case reflect.Ptr:
		if v.IsNil() {
			return payload
		}
		elem := v.Elem()
		if elem.Kind() == reflect.Struct {
			normalized := normalizeStructSlices(elem)
			return normalized
		}
	}
	return payload
}

func normalizeStructSlices(v reflect.Value) any {
	copy := reflect.New(v.Type()).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !copy.Field(i).CanSet() {
			continue
		}
		if field.Kind() == reflect.Slice && field.IsNil() {
			copy.Field(i).Set(reflect.MakeSlice(field.Type(), 0, 0))
			continue
		}
		copy.Field(i).Set(field)
	}
	return copy.Interface()
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}