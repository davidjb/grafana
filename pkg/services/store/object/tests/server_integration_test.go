package object_server_tests

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana/pkg/services/store/object"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func createContentsHash(contents []byte) string {
	hash := md5.Sum(contents)
	return hex.EncodeToString(hash[:])
}

type rawObjectMatcher struct {
	uid           *string
	kind          *string
	createdRange  []time.Time
	modifiedRange []time.Time
	createdBy     *object.UserInfo
	modifiedBy    *object.UserInfo
	body          []byte
	version       *string
	comment       *string
}

func userInfoMatches(expected *object.UserInfo, actual *object.UserInfo) (bool, string) {
	var mismatches []string

	if expected.Id != actual.Id {
		mismatches = append(mismatches, fmt.Sprintf("expected ID %d, actual ID: %d", expected.Id, actual.Id))
	}

	if expected.Login != actual.Login {
		mismatches = append(mismatches, fmt.Sprintf("expected login %s, actual login: %s", expected.Login, actual.Login))
	}

	return len(mismatches) == 0, strings.Join(mismatches, ", ")
}

func timestampInRange(ts int64, tsRange []time.Time) bool {
	return ts >= tsRange[0].Unix() && ts <= tsRange[1].Unix()
}

func requireObjectMatch(t *testing.T, obj *object.RawObject, m rawObjectMatcher) {
	t.Helper()
	mismatches := ""
	if m.uid != nil && *m.uid != obj.UID {
		mismatches += fmt.Sprintf("expected UID: %s, actual UID: %s\n", *m.uid, obj.UID)
	}

	if m.kind != nil && *m.kind != obj.Kind {
		mismatches += fmt.Sprintf("expected kind: %s, actual kind: %s\n", *m.kind, obj.Kind)
	}

	if len(m.createdRange) == 2 && !timestampInRange(obj.Created, m.createdRange) {
		mismatches += fmt.Sprintf("expected createdBy range: [from %s to %s], actual created: %s\n", m.createdRange[0], m.createdRange[1], time.Unix(obj.Created, 0))
	}

	if len(m.modifiedRange) == 2 && !timestampInRange(obj.Modified, m.modifiedRange) {
		mismatches += fmt.Sprintf("expected createdBy range: [from %s to %s], actual created: %s\n", m.createdRange[0], m.createdRange[1], time.Unix(obj.Created, 0))
	}

	if m.createdBy != nil {
		userInfoMatches, msg := userInfoMatches(m.createdBy, obj.CreatedBy)
		if !userInfoMatches {
			mismatches += fmt.Sprintf("createdBy: %s\n", msg)
		}
	}

	if m.modifiedBy != nil {
		userInfoMatches, msg := userInfoMatches(m.modifiedBy, obj.ModifiedBy)
		if !userInfoMatches {
			mismatches += fmt.Sprintf("modifiedBy: %s\n", msg)
		}
	}

	if m.body != nil {
		if !reflect.DeepEqual(m.body, obj.Body) {
			mismatches += fmt.Sprintf("expected body len: %d, actual body len: %d\n", len(m.body), len(obj.Body))
		}

		expectedHash := createContentsHash(m.body)
		actualHash := createContentsHash(obj.Body)
		if expectedHash != actualHash {
			mismatches += fmt.Sprintf("expected body hash: %s, actual body hash: %s\n", expectedHash, actualHash)
		}
	}

	if m.version != nil && *m.version != obj.Version {
		mismatches += fmt.Sprintf("expected version: %s, actual version: %s\n", *m.version, obj.Version)
	}

	if m.comment != nil && *m.comment != obj.Comment {
		mismatches += fmt.Sprintf("expected comment: %s, actual comment: %s\n", *m.comment, obj.Comment)
	}

	require.True(t, len(mismatches) == 0, mismatches)
}

func TestObjectServer(t *testing.T) {
	ctx := context.Background()
	testCtx := createTestContext(t)
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", fmt.Sprintf("Bearer %s", testCtx.authToken))

	fakeUser := &object.UserInfo{
		Login: "fake",
		Id:    1,
	}
	firstVersion := "1"
	kind := "dashboard"
	uid := "my-test-entity"
	body := []byte("{\"name\":\"John\"}")

	t.Run("should not retrieve non-existent objects", func(t *testing.T) {
		resp, err := testCtx.client.Read(ctx, &object.ReadObjectRequest{
			UID:  uid,
			Kind: kind,
		})
		require.NoError(t, err)

		require.NotNil(t, resp)
		require.Nil(t, resp.Object)
	})

	t.Run("should be able to read persisted objects", func(t *testing.T) {
		before := time.Now()
		writeReq := &object.WriteObjectRequest{
			UID:     uid,
			Kind:    kind,
			Body:    body,
			Comment: "first entity!",
		}
		writeResp, err := testCtx.client.Write(ctx, writeReq)
		require.NoError(t, err)

		objectMatcher := rawObjectMatcher{
			uid:           &uid,
			kind:          &kind,
			createdRange:  []time.Time{before, time.Now()},
			modifiedRange: []time.Time{before, time.Now()},
			createdBy:     fakeUser,
			modifiedBy:    fakeUser,
			body:          body,
			version:       &firstVersion,
			comment:       &writeReq.Comment,
		}
		requireObjectMatch(t, writeResp.Object, objectMatcher)

		readResp, err := testCtx.client.Read(ctx, &object.ReadObjectRequest{
			UID:      uid,
			Kind:     kind,
			Version:  "",
			WithBody: true,
		})
		require.NoError(t, err)
		require.Nil(t, readResp.SummaryJson)
		requireObjectMatch(t, writeResp.Object, objectMatcher)

		deleteResp, err := testCtx.client.Delete(ctx, &object.DeleteObjectRequest{
			UID:             uid,
			Kind:            kind,
			PreviousVersion: writeResp.Object.Version,
		})
		require.NoError(t, nil)
		require.True(t, deleteResp.OK)
	})
}
