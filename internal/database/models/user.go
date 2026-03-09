package models

type User struct {
	ID       int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name     string `gorm:"type:varchar(100);not null;uniqueIndex;comment:用户名" json:"name"`
	Password string `gorm:"type:varchar(255);not null;comment:密码" json:"password"`
}

func (User) TableName() string {
	return "user"
}
