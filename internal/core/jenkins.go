package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ZebraOps/ZebraCICD/pkg/log"
)

type JenkinsConfig struct {
	BuildWaitTimeout time.Duration
	PollInterval     time.Duration
}

type JenkinsClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
	config   JenkinsConfig
}

type JenkinsBuildResult struct {
	JobName     string
	BuildNumber int
	QueueID     int
}

type JenkinsBuildStatus struct {
	Number   int    `json:"number"`
	Result   string `json:"result"`   // SUCCESS, FAILURE, ABORTED, null (in progress)
	Building bool   `json:"building"` // true if still building
}

// NewJenkinsClient 创建新的Jenkins客户端
func NewJenkinsClient(baseURL, username, password string) *JenkinsClient {
	return &JenkinsClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		client:   &http.Client{Timeout: 30 * time.Second},
		config: JenkinsConfig{
			BuildWaitTimeout: 2 * time.Minute,
			PollInterval:     5 * time.Second,
		},
	}
}

// NewJenkinsClientWithConfig 创建带自定义配置的Jenkins客户端
func NewJenkinsClientWithConfig(baseURL, username, password string, config JenkinsConfig) *JenkinsClient {
	jc := NewJenkinsClient(baseURL, username, password)
	jc.config = config
	return jc
}

// Authenticate 测试基本认证是否有效
func (jc *JenkinsClient) Authenticate() error {
	url := fmt.Sprintf("%s/api/json", jc.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)

	resp, err := jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}
	log.S().Infof("Jenkins authentication successful")
	return nil
}

// CheckJobExists 检查同名的job是否存在
func (jc *JenkinsClient) CheckJobExists(jobName string) (bool, error) {
	url := fmt.Sprintf("%s/job/%s/api/json", jc.baseURL, jobName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)

	resp, err := jc.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check job existence: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// CreateJob 创建一个Jenkins任务 - 增强版本
func (jc *JenkinsClient) CreateJob(jobName, configXML string) error {
	// ✅ 验证 jobName 合法性
	if jobName == "" {
		return fmt.Errorf("job name cannot be empty")
	}

	// ✅ 验证 XML 非空
	if configXML == "" {
		return fmt.Errorf("job config XML cannot be empty")
	}

	url := fmt.Sprintf("%s/createItem?name=%s", jc.baseURL, url.QueryEscape(jobName))
	req, err := http.NewRequest("POST", url, strings.NewReader(configXML))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)
	req.Header.Set("Content-Type", "application/xml")

	resp, err := jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create job: %v", err)
	}
	defer resp.Body.Close()

	// ✅ 关键改进：捕获错误响应体
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		log.S().Errorf(
			"Jenkins create job failed [%d]\nURL: %s\nJob: %s\nError: %s\nXML:\n%s",
			resp.StatusCode,
			url,
			jobName,
			string(body),
			configXML,
		)

		return fmt.Errorf(
			"create job failed with status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	log.S().Infof("Jenkins job created successfully: %s", jobName)
	return nil
}

// BuildJob 触发Jenkins构建（带参数）
func (jc *JenkinsClient) BuildJob(jobName string, params map[string]string) (*JenkinsBuildResult, error) {

	// 1. 构建参数 URL
	paramStr := encodeParams(params)

	// 2. 发起构建请求
	url := fmt.Sprintf("%s/job/%s/buildWithParameters", jc.baseURL, jobName)
	req, err := http.NewRequest("POST", url, strings.NewReader(paramStr))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := jc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger build: %v", err)
	}
	defer resp.Body.Close()

	// ✅ 接受更多的 2xx 状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("build trigger failed with status: %d", resp.StatusCode)
	}

	// 3. 获取队列 ID
	location := resp.Header.Get("Location")
	queueID := extractQueueID(location)
	if queueID == -1 {
		return nil, fmt.Errorf("failed to extract queue id from location: %s", location)
	}

	log.S().Infof("Build queued with ID: %d", queueID)

	// 4. 等待构建编号分配
	buildNumber, err := jc.waitForBuildNumber(jobName, queueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get build number: %v", err)
	}

	log.S().Infof("Build started: %s #%d", jobName, buildNumber)

	return &JenkinsBuildResult{
		JobName:     jobName,
		QueueID:     queueID,
		BuildNumber: buildNumber,
	}, nil
}

// GetBuildStatus 获取构建状态
func (jc *JenkinsClient) GetBuildStatus(jobName string, buildNumber int) (*JenkinsBuildStatus, error) {
	url := fmt.Sprintf("%s/job/%s/%d/api/json", jc.baseURL, jobName, buildNumber)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)

	resp, err := jc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get build status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get build status failed with status: %d", resp.StatusCode)
	}

	var status JenkinsBuildStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &status, nil
}

// Helper methods

func (jbs *JenkinsBuildStatus) IsComplete() bool {
	return !jbs.Building
}

func (jbs *JenkinsBuildStatus) IsSuccess() bool {
	return jbs.Result == "SUCCESS"
}

func extractQueueID(location string) int {
	re := regexp.MustCompile(`/queue/item/(\d+)/?$`)
	matches := re.FindStringSubmatch(location)
	if len(matches) > 1 {
		if id, err := strconv.Atoi(matches[1]); err == nil {
			return id
		}
	}
	return -1
}

func encodeParams(params map[string]string) string {
	var parts []string
	for key, value := range params {
		parts = append(parts, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(value)))
	}
	return strings.Join(parts, "&")
}

func (jc *JenkinsClient) getBuildNumberFromQueue(queueID int) (int, error) {
	url := fmt.Sprintf("%s/queue/item/%d/api/json", jc.baseURL, queueID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)

	resp, err := jc.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("queue info check failed with status: %d", resp.StatusCode)
	}

	var queueInfo struct {
		Executable struct {
			Number int `json:"number"`
		} `json:"executable"`
		Why string `json:"why"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&queueInfo); err != nil {
		return 0, fmt.Errorf("failed to decode queue info: %v", err)
	}

	if queueInfo.Executable.Number > 0 {
		return queueInfo.Executable.Number, nil
	}

	return 0, fmt.Errorf("build not assigned yet: %s", queueInfo.Why)
}

func (jc *JenkinsClient) waitForBuildNumber(jobName string, queueID int) (int, error) {
	timeout := time.After(jc.config.BuildWaitTimeout)
	ticker := time.NewTicker(jc.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return 0, fmt.Errorf("timeout waiting for build number assignment (limit: %v)", jc.config.BuildWaitTimeout)
		case <-ticker.C:
			buildNumber, err := jc.getBuildNumberFromQueue(queueID)
			if err == nil && buildNumber > 0 {
				return buildNumber, nil
			}
		}
	}
}
