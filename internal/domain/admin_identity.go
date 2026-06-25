package domain

const (
	AdminIdentityActorMaxLength     = 80
	AdminIdentityActorSyntaxMessage = "actor must use 1-80 letters, numbers, dots, underscores, hyphens, or at signs"
)

func ValidAdminIdentityActor(actor string) bool {
	if actor == "" || len(actor) > AdminIdentityActorMaxLength {
		return false
	}
	for _, r := range actor {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '.', '_', '-', '@':
			continue
		default:
			return false
		}
	}
	return true
}

func ReservedAdminIdentityActor(actor string) bool {
	switch actor {
	case "admin-key", "local-dev":
		return true
	default:
		return false
	}
}
