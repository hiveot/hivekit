package bucketstoreapi

import (
	"time"
)

type CursorInfo struct {
	Key string
	// Filter string

	// optional bucket instance this cursor operates on
	// if provided it will be released with the cursor. This allows opening a bucket
	// and automatically closing it after use.
	Bucket IBucket
	// the stored cursor
	Cursor IBucketCursor
	// clientID of the cursor owner
	OwnerID string
	// Optional filter data for use by client while iterating
	FilterData string
	// last use of the cursor
	LastUsed time.Time
	// lifespan of cursor after last use
	Lifespan time.Duration
}

type ICursorCache interface {

	// Add adds a cursor to the tracker and returns its key
	//
	//	clientID identifies the cursor owner
	//	cursor is the object holding the cursor
	//	bucket instance created specifically for this cursor. optional.
	//	filterData is optional data for use by client while iterating.
	//	data optional associated data such as filter specifications
	//	lifespan of an unused cursor, after which it will be deleted. Default 1 minute.
	Add(clientID string,
		cursor IBucketCursor,
		bucket IBucket,
		filterData string,
		lifespan time.Duration) string

	// Get returns the cursor with the given key.
	//
	// An error is returned if the cursor is not found, has expired, or belongs to a different owner.
	//
	//	clientID requesting the cursor. Must match the cursor owner.
	//	cursorKey obtained with Add()
	//	updateLastUsed resets the lifespan of the cursor to start now
	Get(clientID string, cursorKey string, updateLastUsed bool) (
		cursor IBucketCursor, ci *CursorInfo, err error)

	// Release releases the cursor and removes the cursor from the tracker
	// If a bucket was included it will be closed as well.
	Release(clientID string, cursorKey string) error
}
