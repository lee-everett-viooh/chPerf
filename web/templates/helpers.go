package templates

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"chperf.example.com/internal/connections"
)

// statusBadgeClass returns DaisyUI badge classes for run status.
func statusBadgeClass(status string) string {
	switch status {
	case "completed":
		return "badge badge-success"
	case "running":
		return "badge badge-warning"
	case "failed":
		return "badge badge-error"
	default:
		return "badge"
	}
}

// formatFloat formats a float for display.
func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// connectionFormTitle returns the form title (Add vs Edit).
func connectionFormTitle(conn *connections.Connection) string {
	if conn == nil {
		return "Add Connection"
	}
	return "Edit Connection"
}

// connectionFormAction returns the form POST action URL.
func connectionFormAction(conn *connections.Connection) string {
	if conn == nil {
		return "/connections"
	}
	return "/connections/" + strconv.FormatInt(conn.ID, 10)
}

// connectionField returns a connection field value, or empty string if conn is nil.
func connectionField(conn *connections.Connection, field string) string {
	if conn == nil {
		return ""
	}
	switch field {
	case "Name":
		return conn.Name
	case "Host":
		return conn.Host
	case "Database":
		return conn.Database
	case "Username":
		return conn.Username
	case "Password":
		return conn.Password
	default:
		return ""
	}
}

// connectionTLSEnabled returns whether TLS is enabled for the connection.
func connectionTLSEnabled(conn *connections.Connection) bool {
	if conn == nil {
		return false
	}
	return conn.TLSEnabled
}

// connectionTLSBadge returns a badge showing TLS on/off.
func connectionTLSBadge(conn connections.Connection) string {
	if conn.TLSEnabled {
		return "On"
	}
	return "Off"
}

// formatWaitMs formats wait-between-queries milliseconds for display (e.g. 250 -> "0.25s", 1000 -> "1s").
func formatWaitMs(ms int) string {
	if ms == 0 {
		return "0s"
	}
	if ms >= 1000 && ms%1000 == 0 {
		return strconv.Itoa(ms/1000) + "s"
	}
	return fmt.Sprintf("%.2fs", float64(ms)/1000)
}

// connectionPort returns the port as string for the form value.
func connectionPort(conn *connections.Connection) string {
	if conn == nil || conn.Port <= 0 {
		return "9000"
	}
	return strconv.Itoa(conn.Port)
}

// RawJSONScript returns a templ.Component that renders a script tag with raw JSON.
// Escapes </script> to prevent breaking out of the tag.
func RawJSONScript(json string) templ.Component {
	safe := strings.ReplaceAll(json, "</script>", "<\\/script>")
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w, `<script id="chart-data" type="application/json">%s</script>`, safe)
		return err
	})
}
