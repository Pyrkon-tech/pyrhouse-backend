package roles

// Role reprezentuje poziom uprawnień użytkownika
type Role string

const (
	User      Role = "user"
	Moderator Role = "moderator"
	Admin     Role = "admin"
)

// HierarchyLevel określa poziom w hierarchii ról
type HierarchyLevel int

const (
	UserLevel      HierarchyLevel = 1
	ModeratorLevel HierarchyLevel = 2
	AdminLevel     HierarchyLevel = 3
)

// GetHierarchyLevel zwraca poziom hierarchii dla danej roli
func (r Role) GetHierarchyLevel() HierarchyLevel {
	switch r {
	case User:
		return UserLevel
	case Moderator:
		return ModeratorLevel
	case Admin:
		return AdminLevel
	default:
		return UserLevel
	}
}

// HasPermission sprawdza, czy rola ma wymagane uprawnienia
func (r Role) HasPermission(requiredRole Role) bool {
	return r.GetHierarchyLevel() >= requiredRole.GetHierarchyLevel()
}

// IsValid sprawdza, czy rola jest prawidłowa
func (r Role) IsValid() bool {
	switch r {
	case User, Moderator, Admin:
		return true
	default:
		return false
	}
}

// String zwraca stringową reprezentację roli
func (r Role) String() string {
	return string(r)
}
