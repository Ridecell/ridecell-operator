/*
Copyright 2019 Ridecell, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sumologic

// Folder
type FolderListResponse struct {
	StdResp
	Folder
	Children []FolderResponse `json:"children"`
}

// Folder
type FolderResponse struct {
	StdResp
	Name        string   `json:"name"`
	ItemType    string   `json:"itemType"`
	ParentID    string   `json:"parentId"`
	Permissions []string `json:"permissions"`
}

type Folder struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    string `json:"parentId"`
}

// ListPersonalFolder List all personal
func (c *Client) ListPersonalFolder() (*FolderListResponse, error) {
	req, err := c.newRequest("GET", "/api/v2/content/folders/personal", nil)
	var resp FolderListResponse
	if err != nil {
		return &resp, err
	}
	_, err = c.do(req, &resp)
	return &resp, err
}

// GetFolder get FolderListResponse using id
func (c *Client) GetFolder(id string) (*FolderListResponse, error) {
	req, err := c.newRequest("GET", "/api/v2/content/folders/"+id, nil)
	var resp FolderListResponse
	if err != nil {
		return &resp, err
	}
	_, err = c.do(req, &resp)
	return &resp, err
}

// CreateFolder Create folder with given Folder
func (c *Client) CreateFolder(folder Folder) (*FolderListResponse, error) {
	req, err := c.newRequest("POST", "/api/v2/content/folders", folder)
	var resp FolderListResponse
	if err != nil {
		return &resp, err
	}
	_, err = c.do(req, &resp)
	return &resp, err
}
