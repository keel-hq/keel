package http

import (
	"bytes"
	"net/http"

	"net/http/httptest"
	"testing"
)

var fakeGithubPackageWebhook = `{
  "action": "published",
  "registry_package": {
    "id": 35087,
    "name": "server",
    "package_type": "docker",
    "html_url": "https://github.com/DingGGu/UtaiteBOX/packages/35087",
    "created_at": "2019-10-11T18:18:58Z",
    "updated_at": "2019-10-11T18:18:58Z",
    "owner": {
      "login": "DingGGu",
      "id": 2981443,
      "node_id": "MDQ6VXNlcjI5ODE0NDM=",
      "avatar_url": "https://avatars3.githubusercontent.com/u/2981443?v=4",
      "gravatar_id": "",
      "url": "https://api.github.com/users/DingGGu",
      "html_url": "https://github.com/DingGGu",
      "followers_url": "https://api.github.com/users/DingGGu/followers",
      "following_url": "https://api.github.com/users/DingGGu/following{/other_user}",
      "gists_url": "https://api.github.com/users/DingGGu/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/DingGGu/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/DingGGu/subscriptions",
      "organizations_url": "https://api.github.com/users/DingGGu/orgs",
      "repos_url": "https://api.github.com/users/DingGGu/repos",
      "events_url": "https://api.github.com/users/DingGGu/events{/privacy}",
      "received_events_url": "https://api.github.com/users/DingGGu/received_events",
      "type": "User",
      "site_admin": false
    },
    "package_version": {
      "id": 130771,
      "version": "1.2.3",
      "summary": "",
      "body": "",
      "body_html": "",
      "manifest": "{\n   \"schemaVersion\": 2,\n   \"mediaType\": \"application/vnd.docker.distribution.manifest.v2+json\",\n   \"config\": {\n      \"mediaType\": \"application/vnd.docker.container.image.v1+json\",\n      \"size\": 1709,\n      \"digest\": \"sha256:2b94d3d75692e4b04dde5046ad3246fe01cc8889cb641c3e116f10e41c51e164\"\n   },\n   \"layers\": [\n      {\n         \"mediaType\": \"application/vnd.docker.image.rootfs.diff.tar.gzip\",\n         \"size\": 2789669,\n         \"digest\": \"sha256:9d48c3bd43c520dc2784e868a780e976b207cbf493eaff8c6596eb871cbd9609\"\n      },\n      {\n         \"mediaType\": \"application/vnd.docker.image.rootfs.diff.tar.gzip\",\n         \"size\": 350,\n         \"digest\": \"sha256:957045d2b582f07cdc07ebbc7d971239bb7bc19f78216fe547609ff495b007f5\"\n      }\n   ]\n}",
      "html_url": "https://github.com/DingGGu/UtaiteBOX/packages/35087?version=1.2.3",
      "target_commitish": "ts",
      "target_oid": "68d2fd4969f35b650b5863da9220a2561ced6f7b",
      "created_at": "2019-10-11T18:19:06Z",
      "updated_at": "2019-11-01T05:30:31Z",
      "metadata": [

      ],
      "package_files": [
        {
          "download_url": "https://github-production-registry-package-file-4f11e5.s3.amazonaws.com/32367513/0804d380-ec9f-11e9-8306-7f87c59605d3?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAIWNJYAX4CSVEH53A%2F20191101%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20191101T053031Z&X-Amz-Expires=300&X-Amz-Signature=d724dc6416ae277e3754530ac83bd5d33b44ceb5fcd8c0fee41c0fa84df1d942&X-Amz-SignedHeaders=host&actor_id=0&response-content-disposition=filename%3D0804d380-ec9f-11e9-8306-7f87c59605d3&response-content-type=application%2Foctet-stream",
          "id": 448033,
          "name": "0804d380-ec9f-11e9-8306-7f87c59605d3",
          "sha256": "957045d2b582f07cdc07ebbc7d971239bb7bc19f78216fe547609ff495b007f5",
          "sha1": null,
          "md5": null,
          "content_type": "application/octet-stream",
          "state": "uploaded",
          "size": 350,
          "created_at": "2019-10-11T18:18:59Z",
          "updated_at": "2019-10-11T18:19:06Z"
        },
        {
          "download_url": "https://github-production-registry-package-file-4f11e5.s3.amazonaws.com/32367513/0804d380-ec9f-11e9-985b-72c935b667c2?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAIWNJYAX4CSVEH53A%2F20191101%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20191101T053031Z&X-Amz-Expires=300&X-Amz-Signature=eee79a1840857f2a651a9299aee338b0e8994800d39985a2d72a77980d86c813&X-Amz-SignedHeaders=host&actor_id=0&response-content-disposition=filename%3D0804d380-ec9f-11e9-985b-72c935b667c2&response-content-type=application%2Foctet-stream",
          "id": 448034,
          "name": "0804d380-ec9f-11e9-985b-72c935b667c2",
          "sha256": "9d48c3bd43c520dc2784e868a780e976b207cbf493eaff8c6596eb871cbd9609",
          "sha1": null,
          "md5": null,
          "content_type": "application/octet-stream",
          "state": "uploaded",
          "size": 2789669,
          "created_at": "2019-10-11T18:18:59Z",
          "updated_at": "2019-10-11T18:19:06Z"
        },
        {
          "download_url": "https://github-production-registry-package-file-4f11e5.s3.amazonaws.com/32367513/0a672d80-ec9f-11e9-8b8c-6867f9c0ea4e?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAIWNJYAX4CSVEH53A%2F20191101%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20191101T053031Z&X-Amz-Expires=300&X-Amz-Signature=9a0a71caf7b65ec2406ba84aa28552df8b07ca9bad6b0f7091e0beadfa9c11f1&X-Amz-SignedHeaders=host&actor_id=0&response-content-disposition=filename%3D0a672d80-ec9f-11e9-8b8c-6867f9c0ea4e&response-content-type=application%2Foctet-stream",
          "id": 448035,
          "name": "0a672d80-ec9f-11e9-8b8c-6867f9c0ea4e",
          "sha256": "2b94d3d75692e4b04dde5046ad3246fe01cc8889cb641c3e116f10e41c51e164",
          "sha1": null,
          "md5": null,
          "content_type": "application/octet-stream",
          "state": "uploaded",
          "size": 1709,
          "created_at": "2019-10-11T18:19:03Z",
          "updated_at": "2019-10-11T18:19:06Z"
        }
      ],
      "author": {
        "login": "DingGGu",
        "id": 2981443,
        "node_id": "MDQ6VXNlcjI5ODE0NDM=",
        "avatar_url": "https://avatars3.githubusercontent.com/u/2981443?v=4",
        "gravatar_id": "",
        "url": "https://api.github.com/users/DingGGu",
        "html_url": "https://github.com/DingGGu",
        "followers_url": "https://api.github.com/users/DingGGu/followers",
        "following_url": "https://api.github.com/users/DingGGu/following{/other_user}",
        "gists_url": "https://api.github.com/users/DingGGu/gists{/gist_id}",
        "starred_url": "https://api.github.com/users/DingGGu/starred{/owner}{/repo}",
        "subscriptions_url": "https://api.github.com/users/DingGGu/subscriptions",
        "organizations_url": "https://api.github.com/users/DingGGu/orgs",
        "repos_url": "https://api.github.com/users/DingGGu/repos",
        "events_url": "https://api.github.com/users/DingGGu/events{/privacy}",
        "received_events_url": "https://api.github.com/users/DingGGu/received_events",
        "type": "User",
        "site_admin": false
      },
      "installation_command": ""
    },
    "registry": {
      "about_url": "https://help.github.com/about-github-package-registry",
      "name": "GitHub docker registry",
      "type": "docker",
      "url": "https://docker.pkg.github.com/DingGGu/UtaiteBOX",
      "vendor": "GitHub Inc"
    }
  },
  "repository": {
    "id": 32367513,
    "node_id": "MDEwOlJlcG9zaXRvcnkzMjM2NzUxMw==",
    "name": "UtaiteBOX",
    "full_name": "DingGGu/UtaiteBOX",
    "private": true,
    "owner": {
      "login": "DingGGu",
      "id": 2981443,
      "node_id": "MDQ6VXNlcjI5ODE0NDM=",
      "avatar_url": "https://avatars3.githubusercontent.com/u/2981443?v=4",
      "gravatar_id": "",
      "url": "https://api.github.com/users/DingGGu",
      "html_url": "https://github.com/DingGGu",
      "followers_url": "https://api.github.com/users/DingGGu/followers",
      "following_url": "https://api.github.com/users/DingGGu/following{/other_user}",
      "gists_url": "https://api.github.com/users/DingGGu/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/DingGGu/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/DingGGu/subscriptions",
      "organizations_url": "https://api.github.com/users/DingGGu/orgs",
      "repos_url": "https://api.github.com/users/DingGGu/repos",
      "events_url": "https://api.github.com/users/DingGGu/events{/privacy}",
      "received_events_url": "https://api.github.com/users/DingGGu/received_events",
      "type": "User",
      "site_admin": false
    },
    "html_url": "https://github.com/DingGGu/UtaiteBOX",
    "description": "UtaiteBOX",
    "fork": false,
    "url": "https://api.github.com/repos/DingGGu/UtaiteBOX",
    "forks_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/forks",
    "keys_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/keys{/key_id}",
    "collaborators_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/collaborators{/collaborator}",
    "teams_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/teams",
    "hooks_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/hooks",
    "issue_events_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/issues/events{/number}",
    "events_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/events",
    "assignees_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/assignees{/user}",
    "branches_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/branches{/branch}",
    "tags_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/tags",
    "blobs_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/git/blobs{/sha}",
    "git_tags_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/git/tags{/sha}",
    "git_refs_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/git/refs{/sha}",
    "trees_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/git/trees{/sha}",
    "statuses_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/statuses/{sha}",
    "languages_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/languages",
    "stargazers_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/stargazers",
    "contributors_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/contributors",
    "subscribers_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/subscribers",
    "subscription_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/subscription",
    "commits_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/commits{/sha}",
    "git_commits_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/git/commits{/sha}",
    "comments_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/comments{/number}",
    "issue_comment_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/issues/comments{/number}",
    "contents_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/contents/{+path}",
    "compare_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/compare/{base}...{head}",
    "merges_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/merges",
    "archive_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/{archive_format}{/ref}",
    "downloads_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/downloads",
    "issues_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/issues{/number}",
    "pulls_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/pulls{/number}",
    "milestones_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/milestones{/number}",
    "notifications_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/notifications{?since,all,participating}",
    "labels_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/labels{/name}",
    "releases_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/releases{/id}",
    "deployments_url": "https://api.github.com/repos/DingGGu/UtaiteBOX/deployments",
    "created_at": "2015-03-17T02:49:38Z",
    "updated_at": "2019-01-06T12:48:56Z",
    "pushed_at": "2019-08-29T03:07:48Z",
    "git_url": "git://github.com/DingGGu/UtaiteBOX.git",
    "ssh_url": "git@github.com:DingGGu/UtaiteBOX.git",
    "clone_url": "https://github.com/DingGGu/UtaiteBOX.git",
    "svn_url": "https://github.com/DingGGu/UtaiteBOX",
    "homepage": "https://www.utaitebox.com/",
    "size": 113576,
    "stargazers_count": 4,
    "watchers_count": 4,
    "language": "TypeScript",
    "has_issues": true,
    "has_projects": true,
    "has_downloads": true,
    "has_wiki": true,
    "has_pages": false,
    "forks_count": 1,
    "mirror_url": null,
    "archived": false,
    "disabled": false,
    "open_issues_count": 14,
    "license": null,
    "forks": 1,
    "open_issues": 14,
    "watchers": 4,
    "default_branch": "ts"
  },
  "sender": {
    "login": "DingGGu",
    "id": 2981443,
    "node_id": "MDQ6VXNlcjI5ODE0NDM=",
    "avatar_url": "https://avatars3.githubusercontent.com/u/2981443?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/DingGGu",
    "html_url": "https://github.com/DingGGu",
    "followers_url": "https://api.github.com/users/DingGGu/followers",
    "following_url": "https://api.github.com/users/DingGGu/following{/other_user}",
    "gists_url": "https://api.github.com/users/DingGGu/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/DingGGu/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/DingGGu/subscriptions",
    "organizations_url": "https://api.github.com/users/DingGGu/orgs",
    "repos_url": "https://api.github.com/users/DingGGu/repos",
    "events_url": "https://api.github.com/users/DingGGu/events{/privacy}",
    "received_events_url": "https://api.github.com/users/DingGGu/received_events",
    "type": "User",
    "site_admin": false
  }
}`

var fakeGithubContainerRegistryWebhook = `{
  "action": "create",
  "package": {
    "id": 779666,
    "name": "utaitebox-server",
    "namespace": "utaitebox",
    "description": "",
    "ecosystem": "CONTAINER",
    "html_url": "https://github.com/utaitebox/packages/779666",
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "package_version": {
      "id": 1284299,
      "name": "sha256:7d3848ba2f2e7f69bebb4b576e5fad0379b64a0b1512aee6ad0ec9e7c6319fed",
      "description": "",
      "blob_store": "s3",
      "html_url": "https://github.com/utaitebox/packages/779666?version=1284299",
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "container_metadata": {
        "tag": {
          "name": "3.2.1",
          "digest": "sha256:7d3848ba2f2e7f69bebb4b576e5fad0379b64a0b1512aee6ad0ec9e7c6319fed"
        },
        "labels": {
          "description": "",
          "source": "",
          "revision": "",
          "image_url": "",
          "licenses": "",
          "all_labels": {

          }
        },
        "manifest": {
          "digest": "sha256:7d3848ba2f2e7f69bebb4b576e5fad0379b64a0b1512aee6ad0ec9e7c6319fed",
          "media_type": "application/vnd.docker.distribution.manifest.v2+json",
          "uri": "repositories/utaitebox/utaitebox-server/manifests/sha256:7d3848ba2f2e7f69bebb4b576e5fad0379b64a0b1512aee6ad0ec9e7c6319fed",
          "size": 735,
          "config": {
            "digest": "sha256:2b94d3d75692e4b04dde5046ad3246fe01cc8889cb641c3e116f10e41c51e164",
            "media_type": "application/vnd.docker.container.image.v1+json",
            "size": 1709
          },
          "layers": [
            {
              "digest": "sha256:9d48c3bd43c520dc2784e868a780e976b207cbf493eaff8c6596eb871cbd9609",
              "media_type": "application/vnd.docker.image.rootfs.diff.tar.gzip",
              "size": 2789669
            },
            {
              "digest": "sha256:957045d2b582f07cdc07ebbc7d971239bb7bc19f78216fe547609ff495b007f5",
              "media_type": "application/vnd.docker.image.rootfs.diff.tar.gzip",
              "size": 350
            }
          ]
        }
      }
    }
  },
  "organization": {
    "login": "UtaiteBOX",
    "id": 65208347,
    "node_id": "MDEyOk9yZ2FuaXphdGlvbjY1MjA4MzQ3",
    "url": "https://api.github.com/orgs/UtaiteBOX",
    "repos_url": "https://api.github.com/orgs/UtaiteBOX/repos",
    "events_url": "https://api.github.com/orgs/UtaiteBOX/events",
    "hooks_url": "https://api.github.com/orgs/UtaiteBOX/hooks",
    "issues_url": "https://api.github.com/orgs/UtaiteBOX/issues",
    "members_url": "https://api.github.com/orgs/UtaiteBOX/members{/member}",
    "public_members_url": "https://api.github.com/orgs/UtaiteBOX/public_members{/member}",
    "avatar_url": "https://avatars.githubusercontent.com/u/65208347?v=4",
    "description": null
  },
  "sender": {
    "login": "DingGGu",
    "id": 2981443,
    "node_id": "MDQ6VXNlcjI5ODE0NDM=",
    "avatar_url": "https://avatars.githubusercontent.com/u/2981443?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/DingGGu",
    "html_url": "https://github.com/DingGGu",
    "followers_url": "https://api.github.com/users/DingGGu/followers",
    "following_url": "https://api.github.com/users/DingGGu/following{/other_user}",
    "gists_url": "https://api.github.com/users/DingGGu/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/DingGGu/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/DingGGu/subscriptions",
    "organizations_url": "https://api.github.com/users/DingGGu/orgs",
    "repos_url": "https://api.github.com/users/DingGGu/repos",
    "events_url": "https://api.github.com/users/DingGGu/events{/privacy}",
    "received_events_url": "https://api.github.com/users/DingGGu/received_events",
    "type": "User",
    "site_admin": false
  }
}`

func TestGithubPackageWebhookHandler(t *testing.T) {

	fp := &fakeProvider{}
	srv, teardown := NewTestingServer(fp)
	defer teardown()

	req, err := http.NewRequest("POST", "/v1/webhooks/github", bytes.NewBuffer([]byte(fakeGithubPackageWebhook)))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}
	req.Header.Set("X-GitHub-Event", "registry_package")

	//The response recorder used to record HTTP responses
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}

	if len(fp.submitted) != 1 {
		t.Fatalf("unexpected number of events submitted: %d", len(fp.submitted))
	}

	if fp.submitted[0].Repository.Name != "docker.pkg.github.com/DingGGu/UtaiteBOX/server" {
		t.Errorf("expected docker.pkg.github.com/DingGGu/UtaiteBOX/server but got %s", fp.submitted[0].Repository.Name)
	}

	if fp.submitted[0].Repository.Tag != "1.2.3" {
		t.Errorf("expected 1.2.3 but got %s", fp.submitted[0].Repository.Tag)
	}
}

func TestGithubContainerRegistryWebhookHandler(t *testing.T) {

	fp := &fakeProvider{}
	srv, teardown := NewTestingServer(fp)
	defer teardown()

	req, err := http.NewRequest("POST", "/v1/webhooks/github", bytes.NewBuffer([]byte(fakeGithubContainerRegistryWebhook)))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}
	req.Header.Set("X-GitHub-Event", "package_v2")

	//The response recorder used to record HTTP responses
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}

	if len(fp.submitted) != 1 {
		t.Fatalf("unexpected number of events submitted: %d", len(fp.submitted))
	}

	if fp.submitted[0].Repository.Name != "ghcr.io/utaitebox/utaitebox-server" {
		t.Errorf("expected ghcr.io/utaitebox/utaitebox-server but got %s", fp.submitted[0].Repository.Name)
	}

	if fp.submitted[0].Repository.Tag != "3.2.1" {
		t.Errorf("expected 3.2.1 but got %s", fp.submitted[0].Repository.Tag)
	}
}
