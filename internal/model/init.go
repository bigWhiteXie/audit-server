package model

var modelMap = make(map[string]Entity)

func init() {
	modelMap[AuditLogName] = &AuditLog{}
}

func GetModel(name string) Entity {
	return modelMap[name]
}
