package auth

var (
	adminUser string
	adminPass string
	agentUser string
	agentPass string
)

// Initialize sets basic auth credentials used by middleware
func Initialize(adminUsername, adminPassword, agentUsername, agentPassword string) {
	adminUser = adminUsername
	adminPass = adminPassword
	agentUser = agentUsername
	agentPass = agentPassword
}

// ValidateAdmin checks admin credentials (simple equality check)
func ValidateAdmin(user, pass string) bool {
	return user == adminUser && pass == adminPass
}

// ValidateAgent checks agent credentials
func ValidateAgent(user, pass string) bool {
	return user == agentUser && pass == agentPass
}
