package main

func cmdLegal(args []string) {
	if len(args) == 0 {
		errOut("usage: legal licenses|add-license|qa")
	}
	// Legal QA delegates to the RAG system via LLM config.
	// License tracking reuses the license commands in ops.
	switch args[0] {
	case "licenses":
		cmdLicense([]string{"list"})
	case "add-license":
		cmdLicense(append([]string{"add"}, args[1:]...))
	case "expiring":
		cmdLicense([]string{"expiring"})
	case "qa":
		// Placeholder — requires LLM integration
		if len(args) < 2 {
			errOut("usage: legal qa <question>")
		}
		okOut(map[string]interface{}{
			"question": args[1],
			"answer":   "Tinh nang tra loi phap luat can ket noi LLM. Chay: sme-cli config set llm.api_key <key>",
			"note":     "LLM integration required",
		})
	default:
		errOut("unknown legal command: " + args[0])
	}
}
