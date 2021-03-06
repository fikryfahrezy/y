package user

import (
	"database/sql"
	"time"

	pgtypeuuid "github.com/jackc/pgtype/ext/gofrs-uuid"
)

type MemberModel struct {
	IsAdmin           bool
	IsApproved        bool
	Name              string
	OtherPhone        string
	WaPhone           string
	HomestayName      string
	HomestayAddress   string
	ProfilePicUrl     string
	HomestayLatitude  string
	HomestayLongitude string
	Username          string
	Password          string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         sql.NullTime
	Id                pgtypeuuid.UUID
}
