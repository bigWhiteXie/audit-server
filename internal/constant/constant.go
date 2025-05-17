package constant

const (
	USER  = "user"
	ADMIN = "admin"
)

const (
	MAX_PAGE      = 1000
	MAX_PAGE_SIZE = 1000
)

const (
	SchedulePosKey = "schedule:pos:"
)

var ValidFields = map[string]bool{"created_at": true, "action": true, "resource_type": true}
