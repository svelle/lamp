package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// launchInteractiveMode starts the interactive TUI for exploring logs
func launchInteractiveMode(logs []LogEntry) error {
	if len(logs) == 0 {
		return fmt.Errorf("no log entries to display")
	}

	// Sort logs by timestamp
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.Before(logs[j].Timestamp)
	})

	app := tview.NewApplication()

	// Create main layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	
	// Create header
	header := tview.NewTextView().
		SetTextColor(tcell.ColorAqua).
		SetText("Mattermost Log Explorer - Press Ctrl+C to exit, Arrow keys to navigate, Enter to view details").
		SetTextAlign(tview.AlignCenter)
	
	// Create log list
	logList := tview.NewList().
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkBlue)
	
	// Create details view
	details := tview.NewTextView().
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("Log Details")
	
	// Create filter input
	filterInput := tview.NewInputField().
		SetLabel("Filter: ").
		SetFieldWidth(40).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				updateLogList(logList, logs, filterInput.GetText(), details)
			}
		})
	
	// Create status bar
	statusBar := tview.NewTextView().
		SetTextColor(tcell.ColorYellow).
		SetText(fmt.Sprintf("Total logs: %d | Time range: %s to %s", 
			len(logs),
			logs[0].Timestamp.Format("2006-01-02 15:04:05"),
			logs[len(logs)-1].Timestamp.Format("2006-01-02 15:04:05")))
	
	// Add components to layout
	flex.AddItem(header, 1, 1, false).
		AddItem(filterInput, 1, 1, true).
		AddItem(tview.NewFlex().
			AddItem(logList, 0, 2, true).
			AddItem(details, 0, 3, false), 0, 10, false).
		AddItem(statusBar, 1, 1, false)
	
	// Initialize log list
	updateLogList(logList, logs, "", details)
	
	// Set up key handlers
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			// Toggle focus between filter and list
			if filterInput.HasFocus() {
				app.SetFocus(logList)
			} else {
				app.SetFocus(filterInput)
			}
			return nil
		}
		return event
	})
	
	// Run application
	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		return err
	}
	
	return nil
}

// updateLogList refreshes the log list with filtered entries
func updateLogList(list *tview.List, logs []LogEntry, filter string, detailsView *tview.TextView) {
	list.Clear()
	
	filterLower := strings.ToLower(filter)
	var filteredLogs []LogEntry
	
	// Apply filter
	if filter == "" {
		filteredLogs = logs
	} else {
		for _, log := range logs {
			if strings.Contains(strings.ToLower(log.Message), filterLower) ||
			   strings.Contains(strings.ToLower(log.Level), filterLower) ||
			   strings.Contains(strings.ToLower(log.Source), filterLower) {
				filteredLogs = append(filteredLogs, log)
			}
		}
	}
	
	// Add logs to list
	for i, log := range filteredLogs {
		levelColor := getLevelColorName(log.Level)
		timestamp := log.Timestamp.Format("15:04:05")
		
		list.AddItem(
			fmt.Sprintf("[%s]%s[white] [%s] %s", 
				levelColor, 
				log.Level, 
				timestamp,
				truncateString(log.Message, 80)),
			log.Source,
			0,
			func(index int) func() {
				return func() {
					showLogDetails(filteredLogs[index], detailsView)
				}
			}(i),
		)
	}
	
	// Select first item if available
	if list.GetItemCount() > 0 {
		list.SetCurrentItem(0)
		showLogDetails(filteredLogs[0], detailsView)
	} else {
		detailsView.SetText("No matching logs found")
	}
}

// showLogDetails displays detailed information about a log entry
func showLogDetails(log LogEntry, view *tview.TextView) {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("[yellow]Timestamp:[white] %s\n\n", log.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("[yellow]Level:[white] [%s]%s[white]\n\n", getLevelColorName(log.Level), log.Level))
	sb.WriteString(fmt.Sprintf("[yellow]Source:[white] %s\n\n", log.Source))
	
	if log.Caller != "" {
		sb.WriteString(fmt.Sprintf("[yellow]Caller:[white] %s\n\n", log.Caller))
	}
	
	if log.User != "" {
		sb.WriteString(fmt.Sprintf("[yellow]User:[white] %s\n\n", log.User))
	}
	
	sb.WriteString(fmt.Sprintf("[yellow]Message:[white]\n%s\n\n", log.Message))
	
	if log.Details != "" {
		sb.WriteString(fmt.Sprintf("[yellow]Details:[white]\n%s\n", log.Details))
	}
	
	view.SetText(sb.String())
	view.ScrollToBeginning()
}

// getLevelColorName returns the tview color name for a log level
func getLevelColorName(level string) string {
	switch strings.ToUpper(level) {
	case "ERROR", "FATAL", "CRITICAL":
		return "red"
	case "WARN", "WARNING":
		return "yellow"
	case "INFO":
		return "green"
	case "DEBUG":
		return "blue"
	default:
		return "white"
	}
}

// truncateString shortens a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
