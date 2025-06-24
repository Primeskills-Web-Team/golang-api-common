package types

import "time"

// User : user entity from user service
type User struct {
	Id            int         `json:"id"`
	FullName      string      `json:"full_name"`
	Dob           string      `json:"dob"`
	Email         string      `json:"email"`
	Provider      string      `json:"provider"`
	RoleUser      []RoleUser  `json:"role"`
	UserCreator   UserCreator `json:"user_creator"`
	AccountActive string      `json:"account_active"`
	Password      string      `json:"-"`
	RoleId        int         `json:"role_id"`
	CPIsDone      int         `json:"cp_is_done"`
	Picture       string      `json:"picture" gorm:"TEXT"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// RoleUser : role user entity from user service
type RoleUser struct {
	Id        int       `json:"id"`
	UserId    int       `json:"user_id"`
	RoleId    int       `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserCreator : user creator entity from user service
type UserCreator struct {
	Id     int   `json:"id"`
	UserId int   `json:"user_id"`
	User   *User `json:"user"`
}
