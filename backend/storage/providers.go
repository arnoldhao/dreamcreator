package storage

import (
    "bytes"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "go.etcd.io/bbolt"
    "dreamcreator/backend/pkg/logger"
    "go.uber.org/zap"
    "dreamcreator/backend/types"
    "sort"
)

// ProviderRecord 是为了存储层解耦 model 包而定义的镜像结构
// 注意：API 层与服务层请使用各自的结构体，存储层仅负责持久化与取回。

type ProviderRecord struct {
    ID         string        `json:"id"`
    Type       string        `json:"type"`
    Policy     string        `json:"policy"`
    Platform   string        `json:"platform"`
    Name       string        `json:"name"`
    BaseURL    string        `json:"base_url"`
    Region     string        `json:"region"`
    APIKey     string        `json:"api_key"` // 明文存储，注意不要打印日志
    // Vertex AI
    ProjectID    string `json:"project_id"`
    SAEmail      string `json:"sa_email"`
    SAPrivateKey string `json:"sa_private_key"`
    Models     []string      `json:"models"`
    RateLimit  RateLimitRec  `json:"rate_limit"`
    Enabled    bool          `json:"enabled"`
    // Reserved / extended
    AuthMethod       string         `json:"auth_method"`
    APIVersion       string         `json:"api_version"`
    InferenceSummary bool           `json:"inference_summary"`
    APIUsage         map[string]any `json:"api_usage"`
    CreatedAt  time.Time     `json:"created_at"`
    UpdatedAt  time.Time     `json:"updated_at"`
}

type RateLimitRec struct {
    RPM         int `json:"rpm"`
    RPS         int `json:"rps"`
    Burst       int `json:"burst"`
    Concurrency int `json:"concurrency"`
}

// legacy LLMProfileRecord removed (use GlobalProfileRecord instead)

// GlobalProfileRecord mirrors types.GlobalProfile for Bolt storage
type GlobalProfileRecord struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Temperature  float64           `json:"temperature"`
    TopP         float64           `json:"top_p"`
    JSONMode     bool              `json:"json_mode"`
    SysPromptTpl string            `json:"sys_prompt_tpl"`
    MaxTokens    int               `json:"max_tokens"`
    Metadata     map[string]string `json:"metadata"`
    CreatedAt    time.Time         `json:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at"`
}

type ModelsCacheRecord struct {
    ProviderID string    `json:"provider_id"`
    Models     []string  `json:"models"`
    UpdatedAt  time.Time `json:"updated_at"`
}

// ModelMetaRecord 保存单模型的扩展信息（可与 ModelsCacheRecord 配合使用）
type ModelMetaRecord struct {
    ProviderID string        `json:"provider_id"`
    Model      string        `json:"model"`
    Info       types.ModelInfo `json:"info"`
    UpdatedAt  time.Time     `json:"updated_at"`
}

// --- Provider CRUD ---

func (s *BoltStorage) SaveProvider(p *ProviderRecord) error {
    if p == nil || p.ID == "" {
        return fmt.Errorf("provider or id empty")
    }
    p.UpdatedAt = time.Now()
    if p.CreatedAt.IsZero() {
        p.CreatedAt = p.UpdatedAt
    }
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(providersBucket)
        buf, err := json.Marshal(p)
        if err != nil {
            return err
        }
        return b.Put([]byte(p.ID), buf)
    })
}

func (s *BoltStorage) GetProvider(id string) (*ProviderRecord, error) {
    if id == "" {
        return nil, fmt.Errorf("id empty")
    }
    var rec ProviderRecord
    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(providersBucket)
        v := b.Get([]byte(id))
        if v == nil {
            return fmt.Errorf("provider not found: %s", id)
        }
        return json.Unmarshal(v, &rec)
    })
    if err != nil {
        return nil, err
    }
    return &rec, nil
}

func (s *BoltStorage) ListProviders() ([]*ProviderRecord, error) {
    var out []*ProviderRecord
    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(providersBucket)
        return b.ForEach(func(k, v []byte) error {
            var rec ProviderRecord
            if err := json.Unmarshal(v, &rec); err != nil {
                return err
            }
            out = append(out, &rec)
            return nil
        })
    })
    if err != nil {
        return nil, err
    }
    return out, nil
}

func (s *BoltStorage) DeleteProvider(id string) error {
    if id == "" {
        return fmt.Errorf("id empty")
    }
    logger.Info("BoltStorage.DeleteProvider", zap.String("id", id))
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(providersBucket)
        if b == nil { return fmt.Errorf("providers bucket missing") }
        if err := b.Delete([]byte(id)); err != nil { return err }
        logger.Info("BoltStorage.DeleteProvider success", zap.String("id", id))
        return nil
    })
}

// legacy LLM Profile CRUD removed

// --- Global Profiles CRUD ---

func (s *BoltStorage) SaveGlobalProfile(p *GlobalProfileRecord) error {
    if p == nil || strings.TrimSpace(p.ID) == "" {
        return fmt.Errorf("global profile or id empty")
    }
    p.UpdatedAt = time.Now()
    if p.CreatedAt.IsZero() { p.CreatedAt = p.UpdatedAt }
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(globalProfilesBucket)
        buf, err := json.Marshal(p)
        if err != nil { return err }
        return b.Put([]byte(p.ID), buf)
    })
}

func (s *BoltStorage) GetGlobalProfile(id string) (*GlobalProfileRecord, error) {
    if strings.TrimSpace(id) == "" { return nil, fmt.Errorf("id empty") }
    var rec GlobalProfileRecord
    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(globalProfilesBucket)
        v := b.Get([]byte(id))
        if v == nil { return fmt.Errorf("global_profile not found: %s", id) }
        return json.Unmarshal(v, &rec)
    })
    if err != nil { return nil, err }
    return &rec, nil
}

func (s *BoltStorage) ListGlobalProfiles() ([]*GlobalProfileRecord, error) {
    out := []*GlobalProfileRecord{}
    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(globalProfilesBucket)
        return b.ForEach(func(k, v []byte) error {
            var rec GlobalProfileRecord
            if err := json.Unmarshal(v, &rec); err != nil { return err }
            out = append(out, &rec)
            return nil
        })
    })
    if err != nil { return nil, err }
    // stable order by UpdatedAt desc, then name
    sort.SliceStable(out, func(i, j int) bool {
        if !out[i].UpdatedAt.Equal(out[j].UpdatedAt) { return out[i].UpdatedAt.After(out[j].UpdatedAt) }
        return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
    })
    return out, nil
}

func (s *BoltStorage) DeleteGlobalProfile(id string) error {
    if strings.TrimSpace(id) == "" { return fmt.Errorf("id empty") }
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(globalProfilesBucket)
        return b.Delete([]byte(id))
    })
}

// --- Models Cache ---

func (s *BoltStorage) SaveModelsCache(rec *ModelsCacheRecord) error {
    if rec == nil || rec.ProviderID == "" {
        return fmt.Errorf("models cache or provider id empty")
    }
    rec.UpdatedAt = time.Now()
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(modelsCacheBucket)
        buf, err := json.Marshal(rec)
        if err != nil {
            return err
        }
        return b.Put([]byte(rec.ProviderID), buf)
    })
}

func (s *BoltStorage) GetModelsCache(providerID string) (*ModelsCacheRecord, error) {
    var out ModelsCacheRecord
    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(modelsCacheBucket)
        v := b.Get([]byte(providerID))
        if v == nil {
            return fmt.Errorf("models cache not found: %s", providerID)
        }
        return json.Unmarshal(v, &out)
    })
    if err != nil {
        return nil, err
    }
    return &out, nil
}

// --- Models Meta CRUD (optional) ---

func (s *BoltStorage) SaveModelMeta(rec *ModelMetaRecord) error {
    if rec == nil || rec.ProviderID == "" || strings.TrimSpace(rec.Model) == "" {
        return fmt.Errorf("model meta or key empty")
    }
    rec.UpdatedAt = time.Now()
    key := []byte(rec.ProviderID + "::" + rec.Model)
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(modelsMetaBucket)
        buf, err := json.Marshal(rec)
        if err != nil { return err }
        return b.Put(key, buf)
    })
}

func (s *BoltStorage) GetModelMeta(providerID, model string) (*ModelMetaRecord, error) {
    var out ModelMetaRecord
    key := []byte(providerID + "::" + model)
    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(modelsMetaBucket)
        v := b.Get(key)
        if v == nil { return fmt.Errorf("model meta not found: %s/%s", providerID, model) }
        return json.Unmarshal(v, &out)
    })
    if err != nil { return nil, err }
    return &out, nil
}

func (s *BoltStorage) ListModelMeta(providerID string) ([]*ModelMetaRecord, error) {
    out := []*ModelMetaRecord{}
    prefix := []byte(providerID + "::")
    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(modelsMetaBucket)
        c := b.Cursor()
        for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
            var rec ModelMetaRecord
            if err := json.Unmarshal(v, &rec); err != nil { return err }
            out = append(out, &rec)
        }
        return nil
    })
    if err != nil { return nil, err }
    return out, nil
}

// --- Maintenance helpers ---

// ResetLLMData clears providers, global profiles and models cache buckets.
func (s *BoltStorage) ResetLLMData() error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        if err := tx.DeleteBucket(providersBucket); err != nil && err != bbolt.ErrBucketNotFound { return err }
        if err := tx.DeleteBucket(globalProfilesBucket); err != nil && err != bbolt.ErrBucketNotFound { return err }
        if err := tx.DeleteBucket(modelsCacheBucket); err != nil && err != bbolt.ErrBucketNotFound { return err }
        if err := tx.DeleteBucket(modelsMetaBucket); err != nil && err != bbolt.ErrBucketNotFound { return err }
        if _, err := tx.CreateBucketIfNotExists(providersBucket); err != nil { return err }
        if _, err := tx.CreateBucketIfNotExists(globalProfilesBucket); err != nil { return err }
        if _, err := tx.CreateBucketIfNotExists(modelsCacheBucket); err != nil { return err }
        if _, err := tx.CreateBucketIfNotExists(modelsMetaBucket); err != nil { return err }
        return nil
    })
}

// (migrations helpers removed; initialization now depends solely on providers list emptiness)
