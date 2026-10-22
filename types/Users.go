package types

type UserDocument struct {
	UUID     string `bson:"uuid" json:"uuid"`
	Email    string `bson:"email" json:"email"`
	Password string `bson:"password" json:"password"`
	Role     string `bson:"role,omitempty" json:"role"`
}

type UserInfo struct {
	UUID  string `json:"uuid"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type TeacherStudentInfo struct {
	UUID       string `json:"uuid"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	VMCount    int    `json:"vm_count"`
	RunningVMs int    `json:"running_vms"`
	LabCount   int    `json:"lab_count"`
}
