package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/justmiles/go-markdown2confluence/lib"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type confluenceCredentials struct {
	Endpoint    string
	Username    string
	Password    string
	AccessToken string
}

// RunServer starts the MCP server on stdio.
// The server wraps the existing markdown2confluence CLI logic by accepting
// a file path to a markdown file and invoking lib.Markdown2Confluence.
func RunServer(version string) error {
	s := server.NewMCPServer(
		"markdown2confluence",
		version,
		server.WithToolCapabilities(true),
	)

	uploadTool := mcp.NewTool("upload_markdown_to_confluence",
		mcp.WithDescription("Upload markdown content as a Confluence page. Confluence credentials are read from environment variables (CONFLUENCE_USERNAME, CONFLUENCE_PASSWORD, CONFLUENCE_ACCESS_TOKEN, CONFLUENCE_ENDPOINT)."),
		mcp.WithString("space",
			mcp.Required(),
			mcp.Description("Confluence space key where the page will be created"),
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Title of the Confluence page"),
		),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Absolute or relative path to the markdown file to upload"),
		),
		mcp.WithString("parent",
			mcp.Required(),
			mcp.Description("Parent page path (e.g. 'Docs/API') or numeric page ID. In most cases this is required because Confluence spaces typically restrict top-level page creation; without a parent the page may create in the space root."),
		),
		mcp.WithBoolean("hardwraps",
			mcp.Description("Render newlines as <br /> tags"),
		),
		mcp.WithBoolean("debug",
			mcp.Description("Enable debug logging to see image processing details"),
		),
		mcp.WithString("comment",
			mcp.Description("Optional version comment for the Confluence page"),
		),
	)
	getSpaceTool := mcp.NewTool("get_confluence_space_by_page_id",
		mcp.WithDescription("Look up the Confluence space key (namespace) for a given page ID. Use this when the caller knows a Confluence page ID but does not know its space key. The space key is returned from the Confluence API; it must never be guessed. Credentials are read from the same environment variables as upload_markdown_to_confluence."),
		mcp.WithString("page_id",
			mcp.Required(),
			mcp.Description("Numeric Confluence page ID"),
		),
	)
	updateByIDTool := mcp.NewTool("update_confluence_page_by_id",
		mcp.WithDescription("Update an existing Confluence page by its numeric page ID with new markdown content. Unlike upload_markdown_to_confluence which matches by space+title, this tool directly targets a specific page by ID, ensuring the update hits the exact intended page. Confluence credentials are read from environment variables (CONFLUENCE_USERNAME, CONFLUENCE_PASSWORD, CONFLUENCE_ACCESS_TOKEN, CONFLUENCE_ENDPOINT)."),
		mcp.WithString("page_id",
			mcp.Required(),
			mcp.Description("Numeric Confluence page ID of the page to update"),
		),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Absolute or relative path to the markdown file whose content will replace the page body"),
		),
		mcp.WithString("title",
			mcp.Description("Optional new title for the page. If omitted, the existing title is preserved."),
		),
		mcp.WithBoolean("hardwraps",
			mcp.Description("Render newlines as <br /> tags"),
		),
		mcp.WithBoolean("debug",
			mcp.Description("Enable debug logging to see image processing details"),
		),
		mcp.WithString("comment",
			mcp.Description("Optional version comment for the Confluence page update"),
		),
	)

	s.AddTool(uploadTool, handleUploadMarkdown)
	s.AddTool(getSpaceTool, handleGetSpaceByPageID)
	s.AddTool(updateByIDTool, handleUpdatePageByID)

	return server.ServeStdio(s)
}

func handleUploadMarkdown(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	space, err := request.RequireString("space")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	filePath, err := request.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	parent := request.GetString("parent", "")
	hardwraps := request.GetBool("hardwraps", false)
	debug := request.GetBool("debug", false)
	comment := request.GetString("comment", "")

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("unable to resolve file path: %s", err)), nil
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return mcp.NewToolResultError(fmt.Sprintf("markdown file does not exist: %s", absPath)), nil
	}

	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Processing markdown file: %s\n", absPath)
		fmt.Fprintf(os.Stderr, "DEBUG: Title: %s\n", title)
		fmt.Fprintf(os.Stderr, "DEBUG: Space: %s\n", space)
		if parent != "" {
			fmt.Fprintf(os.Stderr, "DEBUG: Parent: %s\n", parent)
		}
	}

	m := lib.Markdown2Confluence{
		Space:          space,
		Title:          title,
		Comment:        comment,
		WithHardWraps:  hardwraps,
		Debug:          debug,
		SourceMarkdown: []string{absPath},
	}
	m.SourceEnvironmentVariables()

	if parent != "" {
		m.Parent = parent
	}

	if err := m.Validate(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Capture stdout so we can return the URL printed by the existing upload logic.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("unable to capture stdout: %s", err)), nil
	}
	os.Stdout = w
	defer func() {
		w.Close()
		os.Stdout = oldStdout
	}()

	errors := m.Run()

	w.Close()
	os.Stdout = oldStdout
	outputBytes, _ := io.ReadAll(r)
	output := strings.TrimSpace(string(outputBytes))

	if len(errors) > 0 {
		var errMsgs []string
		for _, e := range errors {
			errMsgs = append(errMsgs, e.Error())
		}
		return mcp.NewToolResultError(strings.Join(errMsgs, "\n")), nil
	}

	result := "Uploaded successfully"
	if output != "" {
		result += ": " + output
	}
	return mcp.NewToolResultText(result), nil
}

func handleGetSpaceByPageID(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID, err := request.RequireString("page_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return mcp.NewToolResultError("page_id is required"), nil
	}

	creds, err := confluenceCredentialsFromEnv()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	spaceKey, err := fetchSpaceKeyByPageID(creds, pageID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Space key: %s", spaceKey)), nil
}

func handleUpdatePageByID(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID, err := request.RequireString("page_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return mcp.NewToolResultError("page_id is required"), nil
	}

	filePath, err := request.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	title := request.GetString("title", "")
	hardwraps := request.GetBool("hardwraps", false)
	debug := request.GetBool("debug", false)
	comment := request.GetString("comment", "")

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("unable to resolve file path: %s", err)), nil
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return mcp.NewToolResultError(fmt.Sprintf("markdown file does not exist: %s", absPath)), nil
	}

	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Updating page by ID: %s\n", pageID)
		fmt.Fprintf(os.Stderr, "DEBUG: File: %s\n", absPath)
		if title != "" {
			fmt.Fprintf(os.Stderr, "DEBUG: New title: %s\n", title)
		}
	}

	m := lib.Markdown2Confluence{
		Title:          title,
		Comment:        comment,
		WithHardWraps:  hardwraps,
		Debug:          debug,
		PageID:         pageID,
		SourceMarkdown: []string{absPath},
	}
	m.SourceEnvironmentVariables()

	if err := m.Validate(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("unable to capture stdout: %s", err)), nil
	}
	os.Stdout = w
	defer func() {
		w.Close()
		os.Stdout = oldStdout
	}()

	errors := m.Run()

	w.Close()
	os.Stdout = oldStdout
	outputBytes, _ := io.ReadAll(r)
	output := strings.TrimSpace(string(outputBytes))

	if len(errors) > 0 {
		var errMsgs []string
		for _, e := range errors {
			errMsgs = append(errMsgs, e.Error())
		}
		return mcp.NewToolResultError(strings.Join(errMsgs, "\n")), nil
	}

	result := "Updated successfully"
	if output != "" {
		result += ": " + output
	}
	return mcp.NewToolResultText(result), nil
}

func confluenceCredentialsFromEnv() (confluenceCredentials, error) {
	var c confluenceCredentials
	c.Username = os.Getenv("CONFLUENCE_USERNAME")
	c.Password = os.Getenv("CONFLUENCE_PASSWORD")
	c.AccessToken = os.Getenv("CONFLUENCE_ACCESS_TOKEN")
	c.Endpoint = os.Getenv("CONFLUENCE_ENDPOINT")

	if c.Endpoint == "" || c.Endpoint == lib.DefaultEndpoint {
		return c, fmt.Errorf("CONFLUENCE_ENDPOINT environment variable is not set")
	}

	if c.AccessToken == "" {
		if c.Username == "" {
			return c, fmt.Errorf("CONFLUENCE_USERNAME environment variable is not set")
		}
		if c.Password == "" {
			return c, fmt.Errorf("CONFLUENCE_PASSWORD environment variable is not set")
		}
	}

	return c, nil
}

func fetchSpaceKeyByPageID(c confluenceCredentials, pageID string) (string, error) {
	url := strings.TrimSuffix(c.Endpoint, "/") + "/rest/api/content/" + pageID + "?expand=space"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("unable to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	if c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	} else {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Confluence API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Confluence API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Space struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"space"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unable to decode Confluence API response: %w", err)
	}
	if result.Space.Key == "" {
		return "", fmt.Errorf("no space key found for page %s", pageID)
	}

	return result.Space.Key, nil
}
