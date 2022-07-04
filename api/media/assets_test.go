package media

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	iktest "github.com/imagekit-developer/imagekit-go/test"
)

var ctx = context.Background()

var respBody = `[{"fileId":"6283b04dc82abf6294aee010","name":"beauty_of_nature_12_6S7aNLP3-.jpg","filePath":"/beauty_of_nature_12_6S7aNLP3-.jpg","Tags":null,"AITags":null,"versionInfo":{"id":"6283b04dc82abf6294aee010","name":"Version 2"},"isPrivateFile":false,"customCoordinates":null,"url":"https://ik.imagekit.io/dk1m7xkgi/beauty_of_nature_12_6S7aNLP3-.jpg","thumbnail":"https://ik.imagekit.io/dk1m7xkgi/tr:n-ik_ml_thumbnail/beauty_of_nature_12_6S7aNLP3-.jpg","fileType":"image","mime":"image/png","height":133,"Width":200,"size":26509,"hasAlpha":true,"customMetadata":{"price":10},"embeddedMetadata":{"DateCreated":"2022-06-07T15:20:32.104Z","DateTimeCreated":"2022-06-07T15:20:32.105Z","ImageHeight":133,"ImageWidth":200},"createdAt":"2022-05-17T14:25:17.543Z","updatedAt":"2022-06-07T15:20:32.107Z"}]`

var singleAssetResp string

var assetsArr []Asset
var asset Asset
var mediaApi *API

func TestMain(m *testing.M) {
	var err error
	mediaApi, err = NewFromConfiguration(iktest.Cfg)

	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal([]byte(respBody), &assetsArr); err != nil {
		log.Fatal(err)
	}

	singleAssetResp = respBody[1 : len(respBody)-1]
	err = json.Unmarshal([]byte(singleAssetResp), &asset)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

/**
REVIEW-COMMENT

Permuaton combination of on all parameters the SDK supports. Some with empty values, incorrect values and correct values.
See test cases starting here  https://github.com/imagekit-developer/imagekit-nodejs/blob/master/tests/mediaLibrary.js#L807For example:
Pass Tags as an array in SDK and assert that SDK is converting it to comma seperating string in query param.
I see searchQuery=, skip=0, sort=ASC_CREATED in expectedUrl, it is wrong. By default nothign should be passed if user didn't pass any param.
*/
func TestMedia_Assets(t *testing.T) {
	var err error
	var expected = assetsArr
	var expectedUrl = "/files?fileType=all&limit=1000&path=%2F&searchQuery=&skip=0&sort=ASC_CREATED&type=file"

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(200, string(respBody)))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	resp, err := mediaApi.Assets(ctx, AssetsParam{})

	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(resp.Data, expected) {
		t.Errorf("\n%v\n%v\n", resp.Data, expected)
	}

	httpTest.Test(expectedUrl, "GET", nil)
}

func TestMedia_AssetById(t *testing.T) {
	var expected = asset
	var mockBody = respBody[1 : len(respBody)-1]

	var cases = map[string]struct {
		fileId     string
		url        string
		result     Asset
		body       string
		statusCode int
		shouldFail bool
	}{
		"get asset successfully": {
			fileId:     "123",
			url:        "/files/123/details",
			body:       mockBody,
			result:     expected,
			statusCode: 200,
			shouldFail: false,
		},
		"check failure": {
			fileId:     "456",
			url:        "/files/456/details",
			body:       `{"message":"not found"}`,
			result:     Asset{},
			statusCode: 400,
			shouldFail: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			httpTest := iktest.NewHttp(t)

			ts := httptest.NewServer(httpTest.Handler(tc.statusCode, tc.body))
			defer ts.Close()

			mediaApi.Config.API.Prefix = ts.URL + "/"

			resp, err := mediaApi.AssetById(ctx, tc.fileId)

			if tc.shouldFail && err == nil {
				t.Error("expected error")
			}

			if !tc.shouldFail && err != nil {
				t.Error(err)
			}

			if !cmp.Equal(resp.Data, tc.result) {
				t.Errorf("\n%v\n%v\n", resp.Data, expected)
			}

			httpTest.Test(tc.url, "GET", nil)
		})
	}
}

func TestMedia_AssetVersions(t *testing.T) {
	var cases = map[string]struct {
		fileId     string
		versionId  string
		body       string
		statusCode int
		shouldFail bool
	}{
		"all versions": {
			fileId:     "6283b04dc82abf6294aee010",
			versionId:  "v123",
			body:       singleAssetResp,
			statusCode: 200,
			shouldFail: false,
		},
		"should fail": {
			fileId:     "123",
			body:       `{"message": "not found"}`,
			statusCode: 400,
			shouldFail: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var expectedUrl = "/files/" + tc.fileId + "/versions"

			if tc.versionId != "" {
				expectedUrl = expectedUrl + "/" + tc.versionId
			}

			httpTest := iktest.NewHttp(t)

			ts := httptest.NewServer(httpTest.Handler(tc.statusCode, string(tc.body)))
			defer ts.Close()

			mediaApi.Config.API.Prefix = ts.URL + "/"

			params := AssetVersionsParam{
				FileId:    tc.fileId,
				VersionId: tc.versionId,
			}
			_, err := mediaApi.AssetVersions(ctx, params)

			if tc.shouldFail && err == nil {
				t.Error("expected error")
			}

			if !tc.shouldFail && err != nil {
				t.Error(err)
			}

			httpTest.Test(expectedUrl, "GET", nil)
		})
	}
}

/**
REVIEW-COMMENT

Pass all parameters that SDK supports e.g. extensions and customMetadata is missing. See all other as well.
*/
func TestMedia_UpdateAsset(t *testing.T) {
	var expected = asset
	var mockBody = respBody[1 : len(respBody)-1]

	var cases = map[string]struct {
		result     *Asset
		fileId     string
		body       string
		params     UpdateAssetParam
		statusCode int
		shouldFail bool
	}{
		"update asset": {
			result:     &expected,
			fileId:     "file_id",
			body:       mockBody,
			statusCode: 200,
			shouldFail: false,
			params: UpdateAssetParam{
				RemoveAITags:      []string{"one", "two"},
				WebhookUrl:        "http://example.com/hook",
				Tags:              []string{"abc", "def"},
				CustomCoordinates: "12,11,22,22",
			},
		},
		"require fileid": {
			fileId:     "",
			body:       "",
			statusCode: 400,
			shouldFail: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			httpTest := iktest.NewHttp(t)
			ts := httptest.NewServer(httpTest.Handler(200, string(tc.body)))
			defer ts.Close()

			mediaApi.Config.API.Prefix = ts.URL + "/"

			response, err := mediaApi.UpdateAsset(ctx, tc.fileId, tc.params)

			var expectedUrl = "/files/" + tc.fileId + "/details"

			if tc.shouldFail == false {
				httpTest.Test(expectedUrl, "PATCH", tc.params)
				//t.Error("incorrect request body")
			}

			if tc.shouldFail == true && err == nil {
				t.Error("expected err")
			}

			if tc.shouldFail == false && err != nil {
				t.Error("err not nil" + err.Error())
			}

			if !tc.shouldFail && !cmp.Equal(tc.result, &response.Data) {
				t.Errorf("unexpected response %v\n%v", tc.result, response.Data)
			}

		})
	}
}

/**
REVIEW-COMMENT

Negative test case missing. Call AddTags with invalid params e.g. empty/missing FileIds and Tags.
*/
func TestMedia_AddTags(t *testing.T) {
	var err error
	var ids = []string{"xxx", "yyy"}
	var tags = []string{"tag1", "tag2"}
	var resp = UpdatedIds{
		FileIds: ids,
	}

	respBody, _ := json.Marshal(&resp)

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(200, string(respBody)))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	params := TagsParam{
		FileIds: ids,
		Tags:    tags,
	}

	response, err := mediaApi.AddTags(ctx, params)

	if err != nil {
		log.Printf("%+v\n", err)
		t.Errorf("%+v", err)
	}

	if !cmp.Equal(response.Data, resp) {
		t.Errorf("%v\n%v", response.Data, resp)
	}

	var expectedUrl = "/files/addTags"
	httpTest.Test(expectedUrl, "POST", params)
}

/**
REVIEW-COMMENT

Negative test case missing. Call RemoveTags with invalid params e.g. empty/missing FileIds and Tags.
*/
func TestMedia_RemoveTags(t *testing.T) {
	var err error
	var ids = []string{"xxx", "yyy"}
	var tags = []string{"tag1", "tag2"}
	var resp = UpdatedIds{
		FileIds: ids,
	}

	respBody, _ := json.Marshal(&resp)

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(200, string(respBody)))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	params := TagsParam{
		FileIds: ids,
		Tags:    tags,
	}

	response, err := mediaApi.RemoveTags(ctx, params)

	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(response.Data, resp) {
		t.Errorf("%v\n%v", response.Data, resp)
	}

	var expectedUrl = "/files/removeTags"
	httpTest.Test(expectedUrl, "POST", params)
}

/**
REVIEW-COMMENT

Negative test case missing. Call RemoveAITags with invalid params e.g. empty/missing FileIds and AITags.
*/
func TestMedia_RemoveAITags(t *testing.T) {
	var err error
	var ids = []string{"xxx", "yyy"}
	var tags = []string{"tag1", "tag2"}
	var resp = UpdatedIds{
		FileIds: ids,
	}

	respBody, _ := json.Marshal(&resp)

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(200, string(respBody)))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	params := AITagsParam{
		FileIds: ids,
		AITags:  tags,
	}

	response, err := mediaApi.RemoveAITags(ctx, params)

	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(response.Data, resp) {
		t.Errorf("%v\n%v", response.Data, resp)
	}
	httpTest.Test("/files/removeAITags", "POST", params)
}

func TestMedia_DeleteAsset(t *testing.T) {
	var err error

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(200, "1"))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"
	_, err = mediaApi.DeleteAsset(ctx, "file_id")

	if err != nil {
		t.Error(err)
	}

	httpTest.Test("/files/file_id", "DELETE", nil)
}

func TestMedia_DeleteAssetVersion(t *testing.T) {
	var err error

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(204, "1"))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"
	_, err = mediaApi.DeleteAssetVersion(ctx, "file_id", "v2")

	if err != nil {
		t.Error(err)
	}

	url := "/files/file_id/versions/v2"

	httpTest.Test(url, "DELETE", nil)
}

/**
REVIEW-COMMENT

Negative test case missing. Call DeleteBulkAssets with invalid params e.g. empty/missing FileIds.
*/
func TestMedia_DeleteBulkAssets(t *testing.T) {
	var err error
	var param = FileIdsParam{
		FileIds: []string{
			"file_id1", "file_id2",
		},
	}
	var respBody = `{"successfullyDeletedFileIds":["file_id1","file_id2"]}`

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(200, string(respBody)))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	resp, err := mediaApi.DeleteBulkAssets(ctx, param)

	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(resp.Data.FileIds, param.FileIds) {
		t.Errorf("expected: %v, got: %v", param.FileIds, resp.Data.FileIds)
	}
	httpTest.Test("/files/batch/deleteByFileIds", "POST", param)
}

/**
REVIEW-COMMENT

Negative test case missing of invalid/missing params, including non 2xx response from backend.
*/
func TestMedia_CopyAsset(t *testing.T) {
	var err error
	var param = CopyAssetParam{
		SourcePath:          "/file.jpg",
		DestinationPath:     "/natural/file.jpg",
		IncludeFileVersions: true,
	}

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(204, ""))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	/**
	REVIEW-COMMENT

	Please rename CopyAsset to CopyFile, same feedback for moveAsset
	*/
	_, err = mediaApi.CopyAsset(ctx, param)
	if err != nil {
		t.Error(err)
	}

}

/**
REVIEW-COMMENT

Negative test case missing of invalid/missing params, including non 2xx response from backend.
*/
func TestMedia_MoveAsset(t *testing.T) {
	var err error
	var param = MoveAssetParam{
		SourcePath:      "/file.jpg",
		DestinationPath: "/natural/",
	}

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(204, ""))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	_, err = mediaApi.MoveAsset(ctx, param)
	if err != nil {
		t.Error(err)
	}

	httpTest.Test("/files/move", "POST", param)
}

/**
REVIEW-COMMENT

Negative test case missing of invalid/missing params, including non 2xx response from backend.
Also cover calling RenameAsset without PurgeCache and ensure that this parameter is not being sent.
*/
func TestMedia_RenameAsset(t *testing.T) {
	var err error
	var param = RenameAssetParam{
		FilePath:    "/some/file.jpg",
		NewFileName: "/default.jpg",
		PurgeCache:  true,
	}
	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(200, `{"purgeRequestId":"123"}`))

	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	resp, err := mediaApi.RenameAsset(ctx, param)
	if err != nil {
		t.Error(err)
	}

	if resp.Data.RequestId != "123" {
		t.Error("unexpected request id returned")
	}

	httpTest.Test("/files/rename", "PUT", param)
}

/**
REVIEW-COMMENT

Negative test case missing of invalid/missing params, including non 2xx response from backend.
*/
func TestMedia_RestoreVersion(t *testing.T) {
	var err error

	var param = AssetVersionsParam{
		FileId:    "file_id",
		VersionId: "v1",
	}

	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(200, singleAssetResp))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	resp, err := mediaApi.RestoreVersion(ctx, param)
	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(resp.Data, asset) {
		t.Error("unexpected response")
	}

	expectedUrl := fmt.Sprintf("/files/%s/versions/%s/restore",
		param.FileId, param.VersionId)

	httpTest.Test(expectedUrl, "DELETE", param)
}

/**
REVIEW-COMMENT

Negative test case missing of invalid/missing params, including non 2xx response from backend.
*/
func TestMedia_BulkJobStatus(t *testing.T) {
	var err error
	var mockBody = `{"jobId":"job_id","type":"MOVE_FOLDER","status":"Completed"}`
	var res = JobStatusResponse{
		Data: JobStatus{"job_id", "MOVE_FOLDER", "Completed"},
	}
	_ = json.Unmarshal([]byte(mockBody), &res)
	var jobId = "job_id"
	httpTest := iktest.NewHttp(t)

	ts := httptest.NewServer(httpTest.Handler(200, mockBody))
	defer ts.Close()

	mediaApi.Config.API.Prefix = ts.URL + "/"

	resp, err := mediaApi.BulkJobStatus(ctx, jobId)
	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(resp.Data, res.Data) {
		t.Error("unexpected response")
	}

	httpTest.Test("/bulkJobs/"+jobId, "GET", nil)
}
