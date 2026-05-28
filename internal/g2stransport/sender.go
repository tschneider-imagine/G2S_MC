package g2stransport

func NewSender(mode Mode) Sender {
	return &HTTPSender{}
}
