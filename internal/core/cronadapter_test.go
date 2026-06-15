package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRobfigAdapter_Parse_Valid(t *testing.T) {
	a := NewRobfigAdapter()
	sched, err := a.Parse("*/1 * * * *")
	require.NoError(t, err)
	next := sched.Next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, 1, next.Minute())
}

func TestRobfigAdapter_Parse_Invalid(t *testing.T) {
	a := NewRobfigAdapter()
	_, err := a.Parse("not a cron")
	assert.Error(t, err)
}

func TestRobfigAdapter_Parse_Empty(t *testing.T) {
	a := NewRobfigAdapter()
	_, err := a.Parse("")
	assert.Error(t, err)
}

func TestRobfigSchedule_Next_Advances(t *testing.T) {
	a := NewRobfigAdapter()
	sched, err := a.Parse("0 9 * * *")
	require.NoError(t, err)
	from := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
	next := sched.Next(from)
	assert.Equal(t, 9, next.Hour())
	assert.Equal(t, 0, next.Minute())
	assert.True(t, next.After(from))
}
