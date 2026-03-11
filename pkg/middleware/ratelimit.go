package middleware

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	domain "admin-service/internal/domain/model"
	ratelimitrepo "admin-service/internal/domain/rate_limit"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	RateLimitScopeRole = "role"
	RateLimitScopeIP   = "ip"
)

const (
	ResourceUsersCRUD   = "users_crud"
	ResourceThreatsCRUD = "threats_crud"
	ResourceIPGlobal    = "api_ip"
)

type policyConfig struct {
	limit    redis_rate.Limit
	requests int
}

type APIRateLimiter struct {
	limiter    *redis_rate.Limiter
	repo       ratelimitrepo.Repository
	logger     *zap.Logger
	mutex      sync.RWMutex
	roleLimits map[string]map[string]*policyConfig
	ipPolicy   *policyConfig
}

func NewAPIRateLimiter(ctx context.Context, redisClient *redis.Client, repo ratelimitrepo.Repository, logger *zap.Logger) (*APIRateLimiter, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("redis client is required for rate limiting")
	}
	if repo == nil {
		return nil, fmt.Errorf("rate limit repository is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	limiter := redis_rate.NewLimiter(redisClient)
	rl := &APIRateLimiter{
		limiter:    limiter,
		repo:       repo,
		logger:     logger.Named("api-rate-limiter"),
		roleLimits: make(map[string]map[string]*policyConfig),
	}
	if err := rl.reload(ctx); err != nil {
		return nil, err
	}
	return rl, nil
}

// func (l *APIRateLimiter) Reload(ctx context.Context) error {
// 	return l.reload(ctx)
// }

func (l *APIRateLimiter) reload(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	policies, err := l.repo.List(ctx)
	if err != nil {
		return err
	}
	roleLimits := make(map[string]map[string]*policyConfig)
	var ipPolicy *policyConfig
	for _, policy := range policies {
		if policy == nil {
			continue
		}
		cfg := policyToConfig(policy)
		if cfg == nil {
			continue
		}
		switch policy.Scope {
		case RateLimitScopeRole:
			roleName := strings.ToLower(strings.TrimSpace(policy.Role.Name))
			resource := strings.ToLower(strings.TrimSpace(policy.Resource))
			if roleName == "" || resource == "" {
				continue
			}
			if _, ok := roleLimits[roleName]; !ok {
				roleLimits[roleName] = make(map[string]*policyConfig)
			}
			roleLimits[roleName][resource] = cfg
		case RateLimitScopeIP:
			ipPolicy = cfg
		}
	}
	l.mutex.Lock()
	l.roleLimits = roleLimits
	l.ipPolicy = ipPolicy
	l.mutex.Unlock()
	return nil
}

func policyToConfig(policy *domain.RateLimitPolicy) *policyConfig {
	if policy == nil || policy.RequestsPerMinute <= 0 || policy.Burst <= 0 {
		return nil
	}
	limit := redis_rate.Limit{
		Rate:   policy.RequestsPerMinute,
		Burst:  policy.Burst,
		Period: time.Minute,
	}
	return &policyConfig{
		limit:    limit,
		requests: policy.RequestsPerMinute,
	}
}

func (l *APIRateLimiter) Middleware(resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if !l.applyIPLimit(ctx, c) {
			return
		}
		if resource != "" {
			if !l.applyUserLimit(ctx, c, resource) {
				return
			}
		}

		c.Next()
	}
}

func (l *APIRateLimiter) applyIPLimit(ctx context.Context, c *gin.Context) bool {
	policy := l.getIPPolicy()
	if policy == nil {
		return true
	}
	key := fmt.Sprintf("ratelimit:ip:%s", sanitizeKeyPart(c.ClientIP()))
	res, err := l.limiter.Allow(ctx, key, policy.limit)
	if err != nil {
		l.logger.Warn("failed to evaluate ip rate limit", zap.Error(err))
		return true
	}
	l.setRateHeaders(c, policy.requests, max(0, res.Remaining), res.ResetAfter, "IP")
	if res.Allowed == 0 {
		l.respondTooManyRequests(c, res.RetryAfter, "IP")
		return false
	}
	return true
}

func (l *APIRateLimiter) applyUserLimit(ctx context.Context, c *gin.Context, resource string) bool {
	user, ok := AuthUserFromContext(c)
	if !ok || user == nil {
		return true
	}
	policy := l.selectPolicy(resource, user.Roles)
	if policy == nil {
		return true
	}
	key := fmt.Sprintf("ratelimit:user:%s:%s", sanitizeKeyPart(user.ID.String()), sanitizeKeyPart(resource))
	res, err := l.limiter.Allow(ctx, key, policy.limit)
	if err != nil {
		l.logger.Warn("failed to evaluate user rate limit", zap.Error(err))
		return true
	}
	l.setRateHeaders(c, policy.requests, max(0, res.Remaining), res.ResetAfter, "")
	if res.Allowed == 0 {
		l.respondTooManyRequests(c, res.RetryAfter, "user API")
		return false
	}
	return true
}

func (l *APIRateLimiter) selectPolicy(resource string, roles []string) *policyConfig {
	resource = strings.ToLower(strings.TrimSpace(resource))
	if resource == "" {
		return nil
	}
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	var best *policyConfig
	for _, role := range roles {
		key := strings.ToLower(strings.TrimSpace(role))
		if key == "" {
			continue
		}
		rolePolicies, ok := l.roleLimits[key]
		if !ok {
			continue
		}
		policy, ok := rolePolicies[resource]
		if !ok {
			continue
		}
		if best == nil || policy.limit.Rate > best.limit.Rate {
			best = policy
		}
	}
	return best
}

func (l *APIRateLimiter) getIPPolicy() *policyConfig {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.ipPolicy
}

func (l *APIRateLimiter) respondTooManyRequests(c *gin.Context, retry time.Duration, scope string) {
	wait := int(math.Max(1, math.Ceil(retry.Seconds())))
	if retry < 0 {
		wait = 1
	}
	c.Header("Retry-After", strconv.Itoa(wait))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code":    "rate_limit_exceeded",
		"message": fmt.Sprintf("%s rate limit exceeded", scope),
	})
}

func (l *APIRateLimiter) setRateHeaders(c *gin.Context, limit, remaining int, reset time.Duration, suffix string) {
	prefix := "X-RateLimit"
	if suffix != "" {
		prefix = fmt.Sprintf("%s-%s", prefix, suffix)
	}
	if limit < 0 {
		limit = 0
	}
	if remaining < 0 {
		remaining = 0
	}
	resetSeconds := int(math.Ceil(reset.Seconds()))
	if resetSeconds < 0 {
		resetSeconds = 0
	}
	c.Header(fmt.Sprintf("%s-Limit", prefix), strconv.Itoa(limit))
	c.Header(fmt.Sprintf("%s-Remaining", prefix), strconv.Itoa(remaining))
	c.Header(fmt.Sprintf("%s-Reset", prefix), strconv.Itoa(resetSeconds))
}

func sanitizeKeyPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(":", "-", "/", "-", " ", "_")
	return replacer.Replace(value)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
