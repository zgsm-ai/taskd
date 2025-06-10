package utils

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// HINT: For performance consideration, no lock is used. The key is to distinguish alerts. More or less alerts is not a big issue.
var alerts []string

// Log error message and add to alert list
func Errorf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)

	alerts = append(alerts, message)

	logrus.Errorf(format, args...)
}

// Log info message and add to alert list
func Infof(format string, args ...any) {
	message := fmt.Sprintf(format, args...)

	alerts = append(alerts, message)

	logrus.Infof(format, args...)
}

// Log debug message
func Debugf(format string, args ...any) {
	logrus.Debugf(format, args...)
}

// Check if there are any alerts
func HasAlerts() bool {
	return len(alerts) > 0
}

// Report all alerts and clear alert list
func ReportAlerts() error {
	if len(alerts) == 0 {
		return nil
	}
	message := strings.Join(alerts, "\n")
	alerts = []string{}
	ReportError(message)
	return nil
}
