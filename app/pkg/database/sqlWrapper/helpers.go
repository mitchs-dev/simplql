package sqlWrapper

import (
	"regexp"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/data"

	log "github.com/sirupsen/logrus"
)

func extractTableName(query string) string {
	// Split the query into words
	words := strings.Fields(query)

	// Check for different SQL commands and extract the table name accordingly
	switch strings.ToUpper(words[0]) {
	case "SELECT", "DELETE":
		// For SELECT and DELETE, find the table name after the FROM keyword
		for i, word := range words {
			if strings.ToUpper(word) == "FROM" && i+1 < len(words) {
				log.Debug("Returning table name: ", words[i+1])
				return words[i+1]
			}
		}
	case "INSERT":
		// For INSERT INTO, the table name is the third word
		if strings.ToUpper(words[1]) == "INTO" {
			log.Debug("Returning table name: ", words[2])
			return words[2]
		}
	case "UPDATE":
		// For UPDATE, find the table name after the UPDATE keyword
		if len(words) > 1 {
			log.Debug("Returning table name: ", words[1])
		}
		return words[1]
	case "CREATE":
		// For CREATE TABLE, the table name is the third word
		if strings.ToUpper(words[1]) == "TABLE" {
			if len(words) > 4 && strings.ToUpper(words[2]) == "IF" && strings.ToUpper(words[3]) == "NOT" && strings.ToUpper(words[4]) == "EXISTS" {
				log.Debug("Returning table name: ", words[5])
				return words[5] // Adjust index to account for "IF NOT EXISTS"
			}
			log.Debug("Returning table name: ", words[2])
			return words[2]
		}
	default:
		// Return an empty string if the table name could not be determined
		log.Error("Could not determine table name from query: ", query)
		return globals.ErrorTransaction
	}
	return globals.ErrorTransaction
}

// extractRecordID extracts the record ID from a query
func extractRecordID(query string, args []interface{}) string {
	log.Debug("Raw query: ", query)
	log.Debug("Raw args: ", args)
	// Define the regular expression pattern to search for sys_eid
	pattern := `(?i)` + globals.TableEntryIDColumnName + `\s*=\s*['"]?([^'"\s]+)['"]?`
	re := regexp.MustCompile(pattern)

	// Find the first match
	match := re.FindStringSubmatch(query)
	if len(match) > 1 {
		return match[1]
	} else { // If it doesn't match, search in the arguments
		for _, arg := range args {
			// Try to convert the argument to a string
			if argStr, ok := arg.(string); ok {
				if strings.HasPrefix(argStr, globals.EncryptionOriginalFormatHeaderStart) && strings.Contains(argStr, globals.EncryptionOriginalFormatHeaderEnd) {
					log.Debug("Found encrypted arg: ", argStr)
					if !strings.Contains(argStr, globals.EncryptionOriginalFormatHeaderStart+"string"+globals.EncryptionOriginalFormatHeaderEnd) {
						log.Error("Encrypted arg is not of type string")
						return ""
					}
					newArg := data.Process(arg)
					if newArgStr, ok := newArg.(string); ok {
						argStr = newArgStr
					}
				}
				if strings.HasPrefix(argStr, globals.TableEntryIDPrefix) && strings.HasSuffix(argStr, globals.TableEntryIDSuffix) {
					log.Debug("Found entry ID in args: ", argStr)
					return argStr
				} else {
					log.Debug("Not an entry ID: ", argStr)
				}
			}
		}
		log.Warn("Could not find entry ID in query or args")
		return ""
	}
}

func fetchSetColumns(query string) string {
	var oldValueColumns []string

	// Find the SET clause
	setIndex := strings.Index(strings.ToUpper(query), "SET")
	if setIndex == -1 {
		return ""
	}

	// Extract the part of the query after the SET clause
	setClause := query[setIndex+3:]

	// Split the SET clause by commas to get individual assignments
	assignments := strings.Split(setClause, ",")

	for _, assignment := range assignments {
		// Split the assignment by '=' to get the column name
		parts := strings.Split(assignment, "=")
		if len(parts) < 2 {
			continue
		}

		// Trim spaces and get the column name
		column := strings.TrimSpace(parts[0])

		// Add the column to the list of old value columns
		oldValueColumns = append(oldValueColumns, column)
	}

	// Join the old value columns with commas
	return strings.Join(oldValueColumns, ",")
}
