package record

func (r Result) ExitCode(err error) int {
	if err != nil || r.Bytes == 0 {
		return 1
	}
	if r.HadGaps {
		return 2
	}
	return 0
}
