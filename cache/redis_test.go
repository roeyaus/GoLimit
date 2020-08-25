package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTimestampByInterval(t *testing.T) {
	var ts int64
	ts = 1488728901
	newts := GetTimestampByInterval(1, int64(ts))
	assert.Equal(t, ts, newts)

	ts = 1488728880
	newts = GetTimestampByInterval(65, int64(ts))
	assert.Equal(t, ts, newts)

}
