package main

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Projection copied without semantic changes from the #167 input reduction experiment.
var groupingDrop = []string{"old_mode", "new_mode", "head_identity", "worktree_identity", "worktree_kind", "change_hash", "size", "staged", "unstaged"}
var evidenceDrop = []string{"start_line", "end_line", "start_byte", "end_byte"}

// Decode a fresh copy: the immutable Prepared and Git snapshot are never edited.
// Preserve key order as well as all retained JSON values. No reserialization
// of File through a map may introduce a second treatment (property order).
func dropObjectFields(data json.RawMessage, drop []string, evidence bool) (json.RawMessage, int, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, 0, errors.New("invalid file object")
	}
	var out bytes.Buffer
	out.WriteByte('{')
	count, kept := 0, 0
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return nil, 0, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, 0, errors.New("invalid object key")
		}
		var raw json.RawMessage
		if err = d.Decode(&raw); err != nil {
			return nil, 0, err
		}
		remove := false
		for _, name := range drop {
			remove = remove || name == key
		}
		if remove {
			count++
			continue
		}
		if evidence && key == "evidence" {
			var rows []json.RawMessage
			if err = json.Unmarshal(raw, &rows); err != nil {
				return nil, 0, err
			}
			for i, row := range rows {
				value, n, e := dropObjectFields(row, evidenceDrop, false)
				if e != nil {
					return nil, 0, e
				}
				rows[i] = value
				count += n
			}
			raw, err = json.Marshal(rows)
			if err != nil {
				return nil, 0, err
			}
		}
		if kept > 0 {
			out.WriteByte(',')
		}
		encoded, _ := json.Marshal(key)
		out.Write(encoded)
		out.WriteByte(':')
		out.Write(raw)
		kept++
	}
	if _, err = d.Token(); err != nil {
		return nil, 0, err
	}
	out.WriteByte('}')
	return out.Bytes(), count, nil
}
func groupingProjection(input string, treatment, reverse bool) (string, int, string, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(input), &payload); err != nil {
		return "", 0, "", err
	}
	var files []json.RawMessage
	if err := json.Unmarshal(payload["files"], &files); err != nil {
		return "", 0, "", err
	}
	removed := 0
	if treatment {
		for i, file := range files {
			value, n, err := dropObjectFields(file, groupingDrop, true)
			if err != nil {
				return "", 0, "", err
			}
			files[i] = value
			removed += n
		}
	}
	if reverse {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}
	payload["files"], _ = json.Marshal(files)
	canonical, err := json.Marshal(payload)
	if !treatment && !reverse {
		return input, removed, digest(canonical), err
	}
	return string(canonical), removed, digest(canonical), err
}
