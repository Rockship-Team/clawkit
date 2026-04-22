package main

func cmdLegal(args []string) {
	if len(args) == 0 {
		errOut("usage: legal licenses|add-license|expiring|lookup")
	}
	switch args[0] {
	case "licenses":
		cmdLicense([]string{"list"})
	case "add-license":
		cmdLicense(append([]string{"add"}, args[1:]...))
	case "expiring":
		cmdLicense([]string{"expiring"})
	case "lookup":
		if len(args) < 2 {
			errOut("usage: legal lookup <topic>  (topics: notice_period|probation|working_hours|overtime|maternity|annual_leave|termination|minimum_wage|license_fee)")
		}
		cmdLegalLookup(args[1])
	default:
		errOut("unknown legal command: " + args[0])
	}
}

func cmdLegalLookup(topic string) {
	data := loadLaborLaw()
	if data == nil {
		errOut("labor_law_vn.json not found in data dir")
	}
	entry, ok := data[topic]
	if !ok {
		errOut("unknown topic: " + topic)
	}
	okOut(map[string]interface{}{"topic": topic, "info": entry})
}
