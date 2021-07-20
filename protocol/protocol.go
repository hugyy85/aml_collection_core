package protocol

import (
	"reflect"
	"strings"
)

type CreateTaskRequest struct {
	Application  string              `json:"application_id" binding:"required"`
	Priority     string              `json:"priority,omitempty"`
	Ttl          string              `json:"ttl,omitempty"`
	EntityType   string              `json:"entity_type" binding:"required"`
	CheckMethods map[string][]string `json:"check_methods"  binding:"required"`
	Payload      Payload             `json:"payload"  binding:"required"`
}

type ObjectId struct {
	Id string `json:"id"`
}

type Payload struct {
	Inn            string `json:"inn,omitempty" validate:"numeric,min=10,max=12"`
	PassportNumber string `json:"passport_number,omitempty" validate:"numeric,min=6,max=6"`
}

// GetField возвращает атрибут объекта с помощью строкового названия
// Например Payload.GetField("passport_number") == Payload.PassportNumber
func (p *Payload) GetField(field string) (interface{}, bool) {
	field = toCamelCase(field)
	r := reflect.ValueOf(p)
	f := reflect.Indirect(r).FieldByName(field)
	result := f.String()
	fieldExists := true
	if result == "<invalid Value>" {
		fieldExists = false
	}
	return f.String(), fieldExists
}

func toCamelCase(s string) string {
	words := strings.Split(s, "_")
	res := ""
	for _, word := range words {
		res += strings.Title(word)
	}
	return res
}

type Method struct {
	EntityTypes  []string `json:"entity_types"`
	Description  string   `json:"description"`
	RequiredKeys []string `json:"required_keys"`
	ArgsSchema   struct {
	} `json:"args_schema"`
	AdditionalDataSchema struct {
	} `json:"additional_data_schema"`
}
