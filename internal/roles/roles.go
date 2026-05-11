package roles

// Role represents a user's permission level
type Role string

const (
	User       Role = "user"
	Dispatcher Role = "dispatcher"
	Moderator  Role = "moderator"
	Admin      Role = "admin"
)

// HierarchyLevel defines the level in the role hierarchy
type HierarchyLevel int

const (
	UserLevel       HierarchyLevel = 1
	DispatcherLevel HierarchyLevel = 2
	ModeratorLevel  HierarchyLevel = 3
	AdminLevel      HierarchyLevel = 4
)

// GetHierarchyLevel returns the hierarchy level for the given role
func (r Role) GetHierarchyLevel() HierarchyLevel {
	switch r {
	case User:
		return UserLevel
	case Dispatcher:
		return DispatcherLevel
	case Moderator:
		return ModeratorLevel
	case Admin:
		return AdminLevel
	default:
		return UserLevel
	}
}

// HasPermission checks whether the role has the required permissions
func (r Role) HasPermission(requiredRole Role) bool {
	return r.GetHierarchyLevel() >= requiredRole.GetHierarchyLevel()
}

// IsValid checks whether the role is valid
func (r Role) IsValid() bool {
	switch r {
	case User, Dispatcher, Moderator, Admin:
		return true
	default:
		return false
	}
}

// String returns the string representation of the role
func (r Role) String() string {
	return string(r)
}
