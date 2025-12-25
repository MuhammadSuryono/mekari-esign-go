package entity

import "time"

type Profile struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes ProfileAttributes `json:"attributes"`
}

type ProfileAttributes struct {
	ID        string `json:"id"`
	Email     string `json:"email,omitempty"`
	Name      string `json:"full_name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Company   string `json:"company,omitempty"`
	CompanyID string `json:"company_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Role      string `json:"role,omitempty"`

	Quota        *Quota        `json:"balance,omitempty"`
	Subscription *Subscription `json:"subscription,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type Quota struct {
	RemainingEmeterai int `json:"remaining_emeterai_balance"`
	EmeteraiUsage     int `json:"emeterai_usage"`
	GlobalSignDoc     int `json:"global_sign_document"`
	PsreSigning       int `json:"psre_signing"`
	EkycQuota         int `json:"ekyc_quota"`
}

type Subscription struct {
	Plan      string    `json:"plan"`
	Status    string    `json:"status"`
	ExpiredAt time.Time `json:"expired_at"`
}

type ProfileResponse struct {
	Data    *Profile `json:"data"`
	Message string   `json:"message"`
	Status  int      `json:"status"`
}
