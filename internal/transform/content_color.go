package transform

import (
	"cmp"
	"slices"
	"strings"
)

const contentTagCategory = "content"

type contentTagIndex map[int]contentTagIndexEntry

type contentTagIndexEntry struct {
	group TagTypeModel
	tag   TagModel
}

func newContentTagIndex(st *stores) contentTagIndex {
	index := contentTagIndex{}
	for _, tagID := range st.tagIDs {
		tag := st.tagsByID[tagID]
		group, ok := st.tagTypesByID[tag.TagTypeID]
		if !ok {
			continue
		}
		index[tag.ID] = contentTagIndexEntry{group: group, tag: tag}
	}
	return index
}

func contentAccentColor(tagIDs []int, tagIndex contentTagIndex) string {
	tags := resolvedContentTags(tagIDs, tagIndex)
	for _, entry := range tags {
		if entry.group.Category != nil && *entry.group.Category == contentTagCategory && strings.TrimSpace(entry.tag.ColorBackground) != "" {
			return entry.tag.ColorBackground
		}
	}
	for _, entry := range tags {
		if strings.TrimSpace(entry.tag.ColorBackground) != "" {
			return entry.tag.ColorBackground
		}
	}
	return ""
}

func resolvedContentTags(tagIDs []int, tagIndex contentTagIndex) []contentTagIndexEntry {
	seen := map[int]bool{}
	tags := []contentTagIndexEntry{}
	for _, tagID := range tagIDs {
		if seen[tagID] {
			continue
		}
		entry, ok := tagIndex[tagID]
		if !ok {
			continue
		}
		seen[tagID] = true
		tags = append(tags, entry)
	}
	slices.SortStableFunc(tags, func(a, b contentTagIndexEntry) int {
		return cmp.Or(
			cmp.Compare(sortOrderValue(a.group.SortOrder), sortOrderValue(b.group.SortOrder)),
			cmp.Compare(sortOrderValue(a.tag.SortOrder), sortOrderValue(b.tag.SortOrder)),
			cmp.Compare(a.group.ID, b.group.ID),
			cmp.Compare(a.tag.ID, b.tag.ID),
		)
	})
	return tags
}

func applyContentAccentColors(st *stores) {
	tagIndex := newContentTagIndex(st)
	contentColors := map[int]string{}
	for _, contentID := range st.contentIDs {
		content := st.contentByID[contentID]
		content.Color = contentAccentColor(content.TagIDs, tagIndex)
		contentColors[content.ID] = content.Color
		st.contentByID[contentID] = content
	}
	for _, sessionID := range st.sessionIDs {
		session := st.sessionsByID[sessionID]
		session.Color = contentColors[session.ContentID]
		st.sessionsByID[sessionID] = session
	}
}

func sortOrderValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
