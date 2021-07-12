package protocol

type CreateTaskRequest struct {
	Application string `json:"application_id" binding:"required"`
	Priority string `json:"priority,omitempty"`
	Ttl string `json:"ttl,omitempty"`
	EntityType string `json:"entity_type" binding:"required"`
	CheckMethods map[string][]string `json:"check_methods"  binding:"required"`
	Payload Payload `json:"payload"  binding:"required"`
}

type ObjectId struct{
	Id string `json:"id"`
}

type Payload struct {
	Inn string `json:"inn" binding:"numeric,min=9,max=12"`
}


//func (c CreateTaskRequest) ValidateInputData(token string)  {
//	// validate entity_type
//	entityTypes := []string{"individual", "corporate", "entrepreneur"}
//	validTask := false
//	for _, entity := range entityTypes {
//			if c.EntityType == entity {
//				validTask = true
//				break
//		}
//	}
//
//
//	//  validate check_methods - find check_methods from auth token
//
//
//}
