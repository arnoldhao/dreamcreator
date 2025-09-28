package zhconvert

import (
	"bytes"
	"context"
	"dreamcreator/backend/pkg/proxy"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Converter 中文转换器
type Converter struct {
	config       *Config
	proxyManager proxy.ProxyManager
}

// New 创建新的转换器实例
func New(config *Config, proxyManager proxy.ProxyManager) *Converter {
	if config == nil {
		config = DefaultConfig()
	}

	return &Converter{
		config:       config,
		proxyManager: proxyManager,
	}
}

// ConvertSingle 转换单条文本
func (c *Converter) ConvertSingle(text string, converter ConverterType) (converted string, err error) {
	if text == "" {
		return "", nil
	}

	resp, err := c.doRequest(ConvertRequest{
		Text:      text,
		Converter: converter.String(),
	})
	if err != nil {
		return "", err
	}

	converted = resp.Data.Text
	return converted, nil
}

// ConvertMultiple 转换多条文本，保持顺序
func (c *Converter) ConvertMultiple(texts []string, converter ConverterType) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	// 处理空字符串的索引映射
	var nonEmptyTexts []string
	var indexMap []int // 记录非空文本在原数组中的索引

	for i, text := range texts {
		if text != "" {
			nonEmptyTexts = append(nonEmptyTexts, text)
			indexMap = append(indexMap, i)
		}
	}

	// 如果没有非空文本，直接返回空字符串数组
	if len(nonEmptyTexts) == 0 {
		results := make([]string, len(texts))
		return results, nil
	}

	// 使用分隔符合并所有非空文本
	const separator = "\n---ZHCONVERT_SEPARATOR---\n"
	combinedText := strings.Join(nonEmptyTexts, separator)

	// 调用API一次
	convertedText, err := c.ConvertSingle(combinedText, converter)
	if err != nil {
		return nil, fmt.Errorf("failed to convert combined text: %w", err)
	}

	// 拆解结果
	convertedParts := strings.Split(convertedText, separator)

	// 检查拆解结果数量是否匹配
	if len(convertedParts) != len(nonEmptyTexts) {
		// 如果拆解失败，回退到逐个转换
		return c.convertIndividually(texts, converter)
	}

	// 构建最终结果，保持原始顺序
	results := make([]string, len(texts))
	for i, originalIndex := range indexMap {
		results[originalIndex] = convertedParts[i]
	}

	return results, nil
}

// convertIndividually 回退方案：逐个转换（当批量转换失败时使用）
func (c *Converter) convertIndividually(texts []string, converter ConverterType) ([]string, error) {
	results := make([]string, len(texts))

	for i, text := range texts {
		if text == "" {
			results[i] = ""
			continue
		}

		converted, err := c.ConvertSingle(text, converter)
		if err != nil {
			return nil, fmt.Errorf("failed to convert text at index %d: %w", i, err)
		}
		results[i] = converted

		// 简单的延迟避免过于频繁的请求
		if i < len(texts)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return results, nil
}

// GetAllConverterTypes 返回所有可用的转换器类型
func (c *Converter) GetAllConverterTypes() []ConverterType {
	return []ConverterType{
		ZH_CONVERTER_SIMPLIFIED,
		ZH_CONVERTER_TRADITIONAL,
		ZH_CONVERTER_CHINA,
		ZH_CONVERTER_HONGKONG,
		ZH_CONVERTER_TAIWAN,
		ZH_CONVERTER_PINYIN,
		ZH_CONVERTER_BOPOMOFO,
		ZH_CONVERTER_MARS,
		ZH_CONVERTER_WIKI_SIMPLIFIED,
		ZH_CONVERTER_WIKI_TRADITIONAL,
	}
}

// GetSupportedConverters 获取支持的转换器列表
func (c *Converter) GetSupportedConverters() []string {
	types := c.GetAllConverterTypes()
	result := make([]string, 0, len(types))
	for _, t := range types {
		result = append(result, t.String())
	}
	return result
}

// doRequest 执行HTTP请求
func (c *Converter) doRequest(req ConvertRequest) (*ConvertResponse, error) {
	var lastErr error

	for i := 0; i <= c.config.RetryCount; i++ {
		resp, err := c.makeHTTPRequest(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		// 指数退避重试
		if i < c.config.RetryCount {
			time.Sleep(time.Duration(1<<i) * time.Second)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", c.config.RetryCount, lastErr)
}

// makeHTTPRequest 发起HTTP请求
func (c *Converter) makeHTTPRequest(req ConvertRequest) (*ConvertResponse, error) {
	// 添加超时context
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.APIBaseURL+"/convert", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpClient := c.proxyManager.GetHTTPClient()

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result ConvertResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
