package core

import (
	"encoding/json"
	"encoding/xml"
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

// NewJenkinsClient 创建新的Jenkins客户端（使用默认超时配置）
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

// NewJenkinsClientWithTimeout 创建带自定义超时配置的Jenkins客户端
func NewJenkinsClientWithTimeout(baseURL, username, password string, httpTimeout, buildWaitTimeout, pollInterval time.Duration) *JenkinsClient {
	return &JenkinsClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		client:   &http.Client{Timeout: httpTimeout},
		config: JenkinsConfig{
			BuildWaitTimeout: buildWaitTimeout,
			PollInterval:     pollInterval,
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
	apiURL := fmt.Sprintf("%s/api/json", jc.baseURL)
	req, err := http.NewRequest("GET", apiURL, nil)
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
	apiURL := fmt.Sprintf("%s/job/%s/api/json", jc.baseURL, jobName)
	req, err := http.NewRequest("GET", apiURL, nil)
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

	apiURL := fmt.Sprintf("%s/createItem?name=%s", jc.baseURL, url.QueryEscape(jobName))
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(configXML))
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
			apiURL,
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

// UpdateJob 更新一个Jenkins任务的配置XML
func (jc *JenkinsClient) UpdateJob(jobName, configXML string) error {
	if jobName == "" {
		return fmt.Errorf("job name cannot be empty")
	}
	if configXML == "" {
		return fmt.Errorf("job config XML cannot be empty")
	}

	apiURL := fmt.Sprintf("%s/job/%s/config.xml", jc.baseURL, url.QueryEscape(jobName))
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(configXML))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)
	req.Header.Set("Content-Type", "application/xml")

	resp, err := jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update job: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update job failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.S().Infof("Jenkins job config updated successfully: %s", jobName)
	return nil
}

// DeleteJob 删除一个Jenkins任务
func (jc *JenkinsClient) DeleteJob(jobName string) error {
	if jobName == "" {
		return fmt.Errorf("job name cannot be empty")
	}

	apiURL := fmt.Sprintf("%s/job/%s/doDelete", jc.baseURL, url.QueryEscape(jobName))
	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)

	resp, err := jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete job: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		// 404 表示 Jenkins 中已无此任务，不算失败
		if resp.StatusCode == http.StatusNotFound {
			log.S().Infof("Jenkins job %s not found in Jenkins, skip deletion", jobName)
			return nil
		}
		return fmt.Errorf("delete job failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.S().Infof("Jenkins job deleted successfully: %s", jobName)
	return nil
}

// GetConsoleOutput 获取Jenkins构建的控制台输出
func (jc *JenkinsClient) GetConsoleOutput(jobName string, buildNumber int) (string, error) {
	if jobName == "" || buildNumber <= 0 {
		return "", fmt.Errorf("invalid job name or build number")
	}

	apiURL := fmt.Sprintf("%s/job/%s/%d/consoleText", jc.baseURL, url.QueryEscape(jobName), buildNumber)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)

	resp, err := jc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get console output: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get console output failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read console output: %v", err)
	}

	return string(body), nil
}

// BuildJob 触发Jenkins构建（带参数）
func (jc *JenkinsClient) BuildJob(jobName string, params map[string]string) (*JenkinsBuildResult, error) {

	// 1. 构建参数 URL
	paramStr := encodeParams(params)

	// 2. 发起构建请求
	apiURL := fmt.Sprintf("%s/job/%s/buildWithParameters", jc.baseURL, jobName)
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(paramStr))
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
	apiURL := fmt.Sprintf("%s/job/%s/%d/api/json", jc.baseURL, jobName, buildNumber)
	req, err := http.NewRequest("GET", apiURL, nil)
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
	apiURL := fmt.Sprintf("%s/queue/item/%d/api/json", jc.baseURL, queueID)
	req, err := http.NewRequest("GET", apiURL, nil)
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

// ---- Credentials injection via Groovy Script Console ----

// CreateOrUpdateUsernamePasswordCredential 通过 Groovy 脚本控制台创建或更新 UsernamePassword 凭据
// 单层脚本：变量用 groovyEscape 直接插值，避免嵌套 GroovyShell + uberClassLoader（新版 Jenkins 不支持）
func (jc *JenkinsClient) CreateOrUpdateUsernamePasswordCredential(id, username, password, description string) error {
	if id == "" {
		return fmt.Errorf("credential id cannot be empty")
	}

	script := fmt.Sprintf(`
import com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl
import com.cloudbees.plugins.credentials.CredentialsScope
import com.cloudbees.plugins.credentials.domains.Domain
import jenkins.model.Jenkins

def credId = '%s'
def credDesc = '%s'
def credUser = '%s'
def credPass = '%s'

def domain = Domain.global()
def store = Jenkins.instance.getExtensionList('com.cloudbees.plugins.credentials.SystemCredentialsProvider')[0].getStore()
def existing = store.getCredentials(domain).find { it.id == credId }
if (existing != null) { store.removeCredentials(domain, existing) }
store.addCredentials(domain, new UsernamePasswordCredentialsImpl(CredentialsScope.GLOBAL, credId, credDesc, credUser, credPass))
println "credential " + credId + " created"
`, groovyEscape(id), groovyEscape(description), groovyEscape(username), groovyEscape(password))

	return jc.executeGroovyScript(script)
}

// CreateOrUpdateSecretTextCredential 通过 Groovy 脚本控制台创建或更新 SecretText 凭据
// 两阶段策略：先尝试 StringCredentialsImpl，失败则 fallback 到 UsernamePasswordCredentialsImpl
func (jc *JenkinsClient) CreateOrUpdateSecretTextCredential(id, secret, description string) error {
	if id == "" {
		return fmt.Errorf("credential id cannot be empty")
	}

	// Phase 1: 尝试 StringCredentialsImpl（需要 plain-credentials 插件）
	phase1Script := fmt.Sprintf(`
import com.cloudbees.plugins.credentials.CredentialsScope
import com.cloudbees.plugins.credentials.domains.Domain
import org.jenkinsci.plugins.plaincredentials.impl.StringCredentialsImpl
import com.cloudbees.plugins.credentials.Secret
import jenkins.model.Jenkins

def credId = '%s'
def credDesc = '%s'
def credSecret = '%s'

def domain = Domain.global()
def store = Jenkins.instance.getExtensionList('com.cloudbees.plugins.credentials.SystemCredentialsProvider')[0].getStore()
def existing = store.getCredentials(domain).find { it.id == credId }
if (existing != null) { store.removeCredentials(domain, existing) }
store.addCredentials(domain, new StringCredentialsImpl(CredentialsScope.GLOBAL, credId, credDesc, Secret.fromString(credSecret)))
println "credential " + credId + " created as SecretText"
`, groovyEscape(id), groovyEscape(description), groovyEscape(secret))

	err := jc.executeGroovyScript(phase1Script)
	if err == nil {
		log.S().Infof("SecretText credential %s created successfully (plain-credentials plugin available)", id)
		return nil
	}

	// Phase 1 失败，fallback 到 UsernamePassword
	log.S().Infof("SecretText credential creation failed for %s, falling back to UsernamePassword: %v", id, err)

	// Phase 2: UsernamePassword fallback（token 作 password，占位用户名作 username）
	phase2Script := fmt.Sprintf(`
import com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl
import com.cloudbees.plugins.credentials.CredentialsScope
import com.cloudbees.plugins.credentials.domains.Domain
import jenkins.model.Jenkins

def credId = '%s'
def credDesc = '%s'
def credUser = '%s'
def credPass = '%s'

def domain = Domain.global()
def store = Jenkins.instance.getExtensionList('com.cloudbees.plugins.credentials.SystemCredentialsProvider')[0].getStore()
def existing = store.getCredentials(domain).find { it.id == credId }
if (existing != null) { store.removeCredentials(domain, existing) }
store.addCredentials(domain, new UsernamePasswordCredentialsImpl(CredentialsScope.GLOBAL, credId, credDesc + " (token fallback)", credUser + "-token", credPass))
println "credential " + credId + " created as UsernamePassword fallback"
`, groovyEscape(id), groovyEscape(description), groovyEscape(id), groovyEscape(secret))

	return jc.executeGroovyScript(phase2Script)
}

// executeGroovyScript 在 Jenkins 脚本控制台上执行 Groovy 脚本
func (jc *JenkinsClient) executeGroovyScript(script string) error {
	apiURL := fmt.Sprintf("%s/scriptText", jc.baseURL)
	formData := url.Values{}
	formData.Set("script", script)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := jc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute groovy script: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read script response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("groovy script failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 检查输出中是否有异常信号
	// Groovy 脚本成功执行时输出通常是简单的 println 内容（如 "credential xxx created"）
	// 失败时 Jenkins 会返回包含 "Exception:" 的错误信息（编译错误或运行时异常）
	// 注意：不再使用 !Contains("credential") 的排除条件，因为该条件会导致包含 "credential" 关键词的
	// 异常输出被静默忽略（例如编译错误消息中可能同时出现 Exception 和 credential）
	output := string(body)
	if strings.Contains(output, "Exception:") || strings.Contains(output, "ERROR:") {
		return fmt.Errorf("groovy script execution error: %s", output)
	}

	log.S().Infof("Jenkins groovy script executed successfully, output: %s", strings.TrimSpace(output))
	return nil
}

// executeGroovyScriptWithOutput 执行 Groovy 脚本并返回输出内容
// 用于需要读取脚本输出结果的场景（如检查凭证是否存在）
func (jc *JenkinsClient) executeGroovyScriptWithOutput(script string) (string, error) {
	apiURL := fmt.Sprintf("%s/scriptText", jc.baseURL)
	formData := url.Values{}
	formData.Set("script", script)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := jc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute groovy script: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read script response: %v", err)
	}

	output := string(body)
	if resp.StatusCode != http.StatusOK {
		return output, fmt.Errorf("groovy script failed with status %d: %s", resp.StatusCode, output)
	}

	if strings.Contains(output, "Exception:") || strings.Contains(output, "ERROR:") {
		return output, fmt.Errorf("groovy script execution error: %s", output)
	}

	return output, nil
}

// CredentialExists 检查 Jenkins 中是否已存在指定 ID 的凭证
func (jc *JenkinsClient) CredentialExists(id string) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("credential id cannot be empty")
	}

	script := fmt.Sprintf(`
import com.cloudbees.plugins.credentials.domains.Domain
import jenkins.model.Jenkins

def credId = '%s'

def domain = Domain.global()
def store = Jenkins.instance.getExtensionList('com.cloudbees.plugins.credentials.SystemCredentialsProvider')[0].getStore()
def existing = store.getCredentials(domain).find { it.id == credId }
if (existing != null) { println "EXISTS" } else { println "NOT_EXISTS" }
`, groovyEscape(id))

	output, err := jc.executeGroovyScriptWithOutput(script)
	if err != nil {
		return false, fmt.Errorf("failed to check credential existence: %v", err)
	}

	output = strings.TrimSpace(output)
	if strings.Contains(output, "EXISTS") && !strings.Contains(output, "NOT_EXISTS") {
		return true, nil
	}
	return false, nil
}

// JenkinsCredentialItem 表示从 Jenkins 凭据 API 返回的单条凭据元数据
type JenkinsCredentialItem struct {
	ID          string `json:"id"`
	TypeName    string `json:"typeName"`    // 如 "Username with password"
	DisplayName string `json:"displayName"` // 如 "admin/****** (my-id)"
	Description string `json:"description"`
	FullName    string `json:"fullName"`
	// 以下字段仅部分类型有值（Jenkins API 会按类型返回额外字段）
	Username string `json:"username,omitempty"`
	Scope    string `json:"scope,omitempty"`
}

// ListCredentials 通过 Jenkins Credentials Plugin REST API 列出全局凭据
// 调用路径: GET {baseURL}/credentials/store/system/domain/_/api/json?depth=2
func (jc *JenkinsClient) ListCredentials() ([]JenkinsCredentialItem, error) {
	apiURL := fmt.Sprintf("%s/credentials/store/system/domain/_/api/json?depth=2", jc.baseURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.SetBasicAuth(jc.username, jc.password)

	resp, err := jc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("credentials plugin not available or not accessible (404)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list credentials failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Credentials []JenkinsCredentialItem `json:"credentials"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode credentials response: %v", err)
	}

	log.S().Infof("ListCredentials: fetched %d credentials from Jenkins", len(result.Credentials))
	return result.Credentials, nil
}

// groovyEscape 转义 Groovy 字串中的单引号和特殊字符（脚本用单引号定界）
func groovyEscape(s string) string {
	r := strings.NewReplacer(
		"'", "\\'",
		"\\", "\\\\",
	)
	return r.Replace(s)
}

// xmlEscape 转义 XML 特殊字符（用于 generateJobConfig）
func xmlEscape(s string) string {
	var buf strings.Builder
	xml.Escape(&buf, []byte(s))
	return buf.String()
}
