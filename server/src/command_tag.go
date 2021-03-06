package main

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxTagLength = 100
)

// commandTag is sent when a user types the "/tag [tag]" command
//
// Example data:
// {
//   tableID: 123,
//   msg: 'inverted priority finesse',
// }
func commandTag(s *Session, d *CommandData) {
	// Validate that the table exists
	tableID := d.TableID
	var t *Table
	if v, ok := tables[tableID]; !ok {
		s.Warning("Table " + strconv.Itoa(tableID) + " does not exist.")
		return
	} else {
		t = v
	}
	g := t.Game

	if !t.Running {
		s.Warning(ChatCommandNotStartedFail)
		return
	}

	// Sanitize, validate, and normalize the tag
	if v, err := sanitizeTag(d.Msg); err != nil {
		s.Warning(err.Error())
		return
	} else {
		d.Msg = v
	}

	if !t.Replay {
		// Store the tag temporarily until the game ends,
		// at which point we will write it to the database
		g.Tags[d.Msg] = s.UserID()

		// Send them an acknowledgement via private message to avoid spoiling information about the
		// ongoing game
		msg := "Successfully added a tag of \"" + d.Msg + "\"."
		chatServerSendPM(s, msg, d.Room)
		return
	}

	// Get the existing tags from the database
	var tags []string
	if v, err := models.GameTags.GetAll(g.ID); err != nil {
		logger.Error("Failed to get the tags for game ID "+strconv.Itoa(g.ID)+":", err)
		s.Error(DefaultErrorMsg)
		return
	} else {
		tags = v
	}

	// Ensure that this tag does not already exist
	for _, tag := range tags {
		if tag == d.Msg {
			s.Warning("This game has already been tagged with \"" + d.Msg + "\".")
			return
		}
	}

	// Add it to the database
	if err := models.GameTags.Insert(g.ID, s.UserID(), d.Msg); err != nil {
		logger.Error("Failed to insert a tag for game ID "+strconv.Itoa(g.ID)+":", err)
		s.Error(DefaultErrorMsg)
		return
	}

	msg := s.Username() + " has added a game tag of \"" + d.Msg + "\"."
	room := "table" + strconv.Itoa(tableID)
	chatServerSend(msg, room)
}

func sanitizeTag(tag string) (string, error) {
	// Validate tag length
	if len(tag) > MaxTagLength {
		return tag, errors.New("Tags cannot be longer than " + strconv.Itoa(MaxTagLength) + " characters.")
	}

	// Check for valid UTF8
	if !utf8.Valid([]byte(tag)) {
		return tag, errors.New("Tags must contain valid UTF8 characters.") // nolint: golint, stylecheck
	}

	// Replace any whitespace that is not a space with a space
	tag2 := tag
	for _, letter := range tag2 {
		if unicode.IsSpace(letter) && letter != ' ' {
			tag = strings.ReplaceAll(tag, string(letter), " ")
		}
	}

	// Trim whitespace from both sides
	tag = strings.TrimSpace(tag)

	// Validate blank tags
	if tag == "" {
		return tag, errors.New("Tags cannot be blank.") // nolint: golint, stylecheck
	}

	return normalizeString(tag), nil
}
