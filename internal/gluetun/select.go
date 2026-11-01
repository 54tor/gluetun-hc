package gluetun

func SelectSource(mode string, files, api Source) Source {
	switch mode {
	case "api":
		return api
	case "files":
		return files
	default:
		return NewFallback(files, api)
	}
}
