package progress

var (
	spinner    = []rune("⠋⠋⠙⠙⠹⠹⠸⠸⠼⠼⠴⠴⠦⠦⠧⠧⠇⠇⠏⠏")
	spinnerLen = len(spinner)
	spinnerPos = 0
)

func GetMark(s *status) rune {
	if s.offset == s.descriptor.Size {
		return '√'
	}
	spinnerPos = (spinnerPos + 1) % spinnerLen
	return spinner[spinnerPos]
}
