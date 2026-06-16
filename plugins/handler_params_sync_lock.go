package plugins

import (
	"fmt"
	"strings"
	"sync"
	"time"

	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
)

type SyncLock struct {
	templateService.BaseHandler
}

type syncLockEntry struct {
	Key       string
	Token     string
	Owner     string
	ExpireAt  int64
	CreatedAt int64
}

var (
	syncLockMutex       sync.Mutex
	syncLockMap         = map[string]*syncLockEntry{}
	syncLockCleanerOnce sync.Once
)

const (
	defaultSyncLockTTLMillis int64 = 10 * 1000
	syncLockCleanupInterval        = time.Minute
)

func (si *SyncLock) ensureCleaner() {
	syncLockCleanerOnce.Do(func() {
		ticker := time.NewTicker(syncLockCleanupInterval)
		go func() {
			for range ticker.C {
				si.cleanupExpired(time.Now().UnixMilli())
			}
		}()
	})
}

func (si *SyncLock) cleanupExpired(nowMillis int64) {
	syncLockMutex.Lock()
	defer syncLockMutex.Unlock()
	for key, item := range syncLockMap {
		if item == nil || item.ExpireAt <= nowMillis {
			delete(syncLockMap, key)
		}
	}
}

func renderParamValue(value interface{}, params map[string]interface{}) string {
	return strings.TrimSpace(gocast.ToString(utils.RenderVarOrValue(value, params)))
}

func (si *SyncLock) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	si.ensureCleaner()

	params := template.GetParams()
	lockKey := renderParamValue(handlerParam.Field, params)
	if utils.IsValueEmpty(lockKey) {
		return common.NotOk("sync_lock field不能为空")
	}

	operation := strings.ToLower(renderParamValue(handlerParam.Operation, params))
	if utils.IsValueEmpty(operation) {
		operation = "acquire"
	}

	token := renderParamValue(handlerParam.Value, params)
	owner := renderParamValue(handlerParam.Right, params)
	ttlMillis := handlerParam.Second
	if ttlMillis <= 0 {
		ttlMillis = defaultSyncLockTTLMillis
	}

	nowMillis := time.Now().UnixMilli()
	si.cleanupExpired(nowMillis)

	switch operation {
	case "acquire":
		syncLockMutex.Lock()
		defer syncLockMutex.Unlock()

		if exists, ok := syncLockMap[lockKey]; ok && exists != nil && exists.ExpireAt > nowMillis {
			left := exists.ExpireAt - nowMillis
			return common.NotOk(fmt.Sprintf("同步任务正在执行中，请稍后重试（lock=%s, owner=%s, left_ms=%d）", lockKey, exists.Owner, left))
		}

		if utils.IsValueEmpty(token) {
			token = fmt.Sprintf("%s-%d", lockKey, time.Now().UnixNano())
		}
		if utils.IsValueEmpty(owner) {
			owner = gocast.ToString(params["service"])
		}

		entry := &syncLockEntry{
			Key:       lockKey,
			Token:     token,
			Owner:     owner,
			ExpireAt:  nowMillis + ttlMillis,
			CreatedAt: nowMillis,
		}
		syncLockMap[lockKey] = entry
		return common.Ok(map[string]interface{}{
			"acquired":   true,
			"lock_key":   entry.Key,
			"lock_token": entry.Token,
			"lock_owner": entry.Owner,
			"expire_at":  entry.ExpireAt,
			"ttl_ms":     ttlMillis,
		}, "同步锁获取成功")

	case "release":
		syncLockMutex.Lock()
		defer syncLockMutex.Unlock()

		exists, ok := syncLockMap[lockKey]
		if !ok || exists == nil {
			return common.Ok(map[string]interface{}{
				"released": false,
				"lock_key": lockKey,
				"reason":   "not_found",
			}, "同步锁不存在，跳过释放")
		}

		if !utils.IsValueEmpty(token) && exists.Token != token {
			return common.Ok(map[string]interface{}{
				"released":      false,
				"lock_key":      lockKey,
				"reason":        "token_mismatch",
				"current_token": exists.Token,
			}, "锁token不匹配，跳过释放")
		}

		delete(syncLockMap, lockKey)
		return common.Ok(map[string]interface{}{
			"released": true,
			"lock_key": lockKey,
		}, "同步锁释放成功")

	default:
		return common.NotOk("sync_lock operation仅支持 acquire/release")
	}
}
