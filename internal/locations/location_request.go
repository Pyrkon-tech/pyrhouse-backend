package locations

type UpdateLocationRequest struct {
	Name     *string `json:"name,omitempty"`
	Details  *string `json:"details,omitempty"`
	Pavilion *string `json:"pavilion,omitempty"`
}
