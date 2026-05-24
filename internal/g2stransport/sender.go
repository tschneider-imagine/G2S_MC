package g2stransport

func NewSender(mode Mode) Sender {
	switch normalizeMode(mode) {
	case ModeHTTP:
		return &HTTPSender{}
	case ModeDryRun, ModeDisabled:
		return &DisabledSender{}
	default:
		return &DisabledSender{}
	}
}
