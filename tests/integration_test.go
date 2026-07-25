package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/rest"
	"github.com/DimKa163/goph-profile/tests/testutil"
	"github.com/stretchr/testify/require"
)

func TestUploadShouldBeSuccess(t *testing.T) {
	server := testutil.NewServer(t)
	t.Cleanup(server.Close)
	userID := "user@user.ru"
	cases := []struct {
		Name               string
		Path               string
		ExpectedStatusCode int
	}{
		{
			Name:               "png should be successful",
			Path:               "testavatar.png",
			ExpectedStatusCode: http.StatusCreated,
		},
		{
			Name:               "jpeg should be successful",
			Path:               "testavatar.jpeg",
			ExpectedStatusCode: http.StatusCreated,
		},
		{
			Name:               "jpg should be successful",
			Path:               "testavatar.jpg",
			ExpectedStatusCode: http.StatusCreated,
		},
		{
			Name:               "webp should be successful",
			Path:               "testavatar.webp",
			ExpectedStatusCode: http.StatusCreated,
		},
		{
			Name:               "gif should be failed",
			Path:               "testavatar.gif",
			ExpectedStatusCode: http.StatusBadRequest,
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			resp, err := testutil.UploadAvatar(t, server, userID, c.Path)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })
			require.Equal(t, c.ExpectedStatusCode, resp.StatusCode)
			data, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if resp.StatusCode == http.StatusCreated {
				var uploadResponse rest.UploadResponse
				err = json.Unmarshal(data, &uploadResponse)
				require.NoError(t, err)

				require.Equal(t, userID, uploadResponse.UserID)
				require.Equal(t, "processing", uploadResponse.Status)
				require.NotEmpty(t, uploadResponse.CreatedAt)
			} else {
				var errorResponse rest.ServiceError
				err = json.Unmarshal(data, &errorResponse)
				require.NoError(t, err)
				require.Equal(t, errorResponse.Code, entity.InvalidContentTypeErrorCode.String())
			}
		})
	}
}

func TestUploadLargeFileShouldBeFailed(t *testing.T) {
	server := testutil.NewServer(t)
	t.Cleanup(server.Close)

	userID := "user@user.ru"
	resp, err := testutil.UploadAvatar(t, server, userID, "testavatar_over_12mb.png")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var errorResponse rest.ServiceError
	err = json.Unmarshal(data, &errorResponse)
	require.NoError(t, err)

	require.Equal(t, errorResponse.Code, entity.InvalidSizeErrorCode.String())
}

func TestUploadWithoutUserIDShouldBeFailed(t *testing.T) {
	server := testutil.NewServer(t)
	t.Cleanup(server.Close)

	userID := ""
	resp, err := testutil.UploadAvatar(t, server, userID, "testavatar.png")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var errorResponse rest.ServiceError
	err = json.Unmarshal(data, &errorResponse)
	require.NoError(t, err)

	require.Equal(t, errorResponse.Code, entity.InvalidUserIDErrorCode.String())
}

func TestUploadWithIncorrectUserIDShouldBeFailed(t *testing.T) {
	server := testutil.NewServer(t)
	t.Cleanup(server.Close)

	userID := "incorrect-user-id"
	resp, err := testutil.UploadAvatar(t, server, userID, "testavatar.png")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var errorResponse rest.ServiceError
	err = json.Unmarshal(data, &errorResponse)
	require.NoError(t, err)

	require.Equal(t, errorResponse.Code, entity.InvalidUserIDErrorCode.String())
}
