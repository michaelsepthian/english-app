// MIVA-DOMAIN: learning data and pure validation rules.
package model

type Message struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}
