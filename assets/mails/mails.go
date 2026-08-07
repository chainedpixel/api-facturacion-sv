package mails

import "embed"

//go:embed layout.html
//go:embed contingency_activated.html
//go:embed emission_failure.html
//go:embed retransmission_job_failed.html
//go:embed partials/icon_warning.svg
//go:embed partials/icon_error.svg
//go:embed partials/icon_info.svg
//go:embed plain/contingency_activated.txt
//go:embed plain/emission_failure.txt
//go:embed plain/retransmission_job_failed.txt
var FS embed.FS
