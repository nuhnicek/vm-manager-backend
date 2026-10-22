package teacher

import (
	"testing"
	"vm-manager-backend/types"
)

func TestCreateStudentValidation(t *testing.T) {
	conn := types.Conn{}

	_, err := CreateStudent(conn, "", "password", "student")
	if err == nil {
		t.Fatal("expected error for empty email, got nil")
	}

	_, err = CreateStudent(conn, "student@test.com", "", "student")
	if err == nil {
		t.Fatal("expected error for empty password, got nil")
	}
}

func TestTeacherStudentInfoModel(t *testing.T) {
	info := types.TeacherStudentInfo{
		UUID:       "uuid-123",
		Email:      "student1@school.cz",
		Role:       "student",
		VMCount:    3,
		RunningVMs: 2,
		LabCount:   1,
	}

	if info.Email != "student1@school.cz" || info.Role != "student" {
		t.Errorf("unexpected student info values: %+v", info)
	}
	if info.VMCount != 3 || info.RunningVMs != 2 {
		t.Errorf("unexpected vm counts: %+v", info)
	}
}
