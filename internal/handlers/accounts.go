package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/exotel"
	"exotel-monitoring-platform/internal/logger"
	"exotel-monitoring-platform/internal/models"
	"exotel-monitoring-platform/internal/repository"
	"exotel-monitoring-platform/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type exophoneInput struct {
	Number   string `json:"number"   binding:"required"`
	Priority string `json:"priority"` // P0–P3; defaults to P1
}

type CreateAccountRequest struct {
	AccountName string          `json:"account_name" binding:"required"`
	SID         string          `json:"sid"          binding:"required"`
	APIKey      string          `json:"api_key"      binding:"required"`
	APIToken    string          `json:"api_token"    binding:"required"`
	Subdomain   string          `json:"subdomain"    binding:"required"`
	Cluster     string          `json:"cluster"`
	Exophones   []exophoneInput `json:"exophones"    binding:"required,min=1"`
}

// fallbackFrequency is used only when priority_config table is unreachable.
var fallbackFrequency = map[string]int{"P0": 15, "P1": 30, "P2": 60, "P3": 60}

const defaultRetryPolicy = `{"max_retries":3,"base_delay_ms":1000,"max_delay_ms":30000}`

// CreateAccount onboards a new organisation atomically:
// account row → exophone rows → 3 monitoring jobs per exophone.
// POST /api/v1/accounts
func CreateAccount(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn("create account: bad request",
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %s", err.Error()),
		})
		return
	}

	cluster := req.Cluster
	if cluster == "" {
		cluster = "in1"
	}

	// Validate each submitted exophone belongs to this Exotel account.
	validNumbers, err := exotel.ListIncomingPhoneNumbers(req.SID, req.Subdomain, req.APIKey, req.APIToken)
	if err != nil {
		logger.Log.Error("create account: exotel validation failed",
			zap.String("sid", req.SID),
			zap.Error(err),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("exotel credential check failed: %s", err.Error())})
		return
	}
	// Normalise submitted numbers and validate against Exotel's set.
	// validNumbers keys are already in canonical format from ListIncomingPhoneNumbers.
	var rejected []string
	for _, ep := range req.Exophones {
		if _, ok := validNumbers[utils.NormalizePhone(ep.Number)]; !ok {
			rejected = append(rejected, ep.Number)
		}
	}
	if len(rejected) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":            "one or more exophone numbers do not belong to this Exotel account",
			"rejected_numbers": rejected,
		})
		return
	}

	// Load frequency config from DB; fall back to hardcoded defaults if unavailable.
	freqMap, freqErr := repository.GetPriorityConfigMap()
	if freqErr != nil {
		logger.Log.Warn("create account: could not load priority config, using defaults", zap.Error(freqErr))
		freqMap = fallbackFrequency
	}

	encryptedKey, err := utils.Encrypt(req.APIKey, config.App.Crypto.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt API key"})
		return
	}
	encryptedToken, err := utils.Encrypt(req.APIToken, config.App.Crypto.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt API token"})
		return
	}

	var (
		createdAccountID uint64
		exophonesCreated int
		jobsCreated      int
	)

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Account
		account := models.Account{
			AccountName: req.AccountName,
			SID:         req.SID,
			APIKey:      encryptedKey,
			APIToken:    encryptedToken,
			Subdomain:   req.Subdomain,
			Cluster:     cluster,
			ConfigJSON:  "{}",
			IsActive:    1,
			CreatedBy:   "API",
			UpdatedBy:   "API",
		}
		if err := tx.Create(&account).Error; err != nil {
			return fmt.Errorf("account insert failed: %w", err)
		}
		createdAccountID = account.ID

		// 2. Exophones + monitoring jobs
		for _, ep := range req.Exophones {
			priority := strings.ToUpper(ep.Priority)
			if _, ok := freqMap[priority]; !ok {
				priority = "P1"
			}

			exophone := models.Exophone{
				AccountID:        account.ID,
				ExophoneNumber:   utils.NormalizePhone(ep.Number),
				Status:           "active",
				Priority:         priority,
				MonitoringFlag:   1,
				IsCronApplicable: 1,
				CacheTTL:         900,
				MetadataJSON:     "{}",
				CreatedBy:        "API",
				UpdatedBy:        "API",
			}
			if err := tx.Create(&exophone).Error; err != nil {
				return fmt.Errorf("exophone insert failed for %s: %w", ep.Number, err)
			}
			exophonesCreated++

			digits := utils.NormalizePhone(ep.Number)
			freq := freqMap[priority]

			for _, jt := range []struct {
				prefix  string
				jobType string
				timeout int
			}{
				{"HB", "HEARTBEAT", 30},
				{"CALLS", "CALLS", 60},
				{"STRM", "STREAMS", 30},
			} {
				job := models.MonitoringJob{
					ExophoneID:      exophone.ID,
					AccountID:       account.ID,
					JobName:         fmt.Sprintf("%s-%s-%s", jt.prefix, priority, digits),
					JobType:         jt.jobType,
					Priority:        priority,
					FrequencyMinute: freq,
					TimeoutSeconds:  jt.timeout,
					RetryPolicy:     defaultRetryPolicy,
					NextRunAt:       time.Now(),
					LastStatus:      "PENDING",
					IsActive:        1,
					CreatedBy:       "API",
					UpdatedBy:       "API",
				}
				if err := tx.Create(&job).Error; err != nil {
					return fmt.Errorf("job insert failed for %s/%s: %w", ep.Number, jt.jobType, err)
				}
				jobsCreated++
			}
		}
		return nil
	})

	if txErr != nil {
		logger.Log.Error("create account: transaction rolled back",
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(txErr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("organisation creation failed: %s", txErr.Error()),
		})
		return
	}

	logger.Log.Info("create account: success",
		zap.Uint64("account_id", createdAccountID),
		zap.Int("exophones", exophonesCreated),
		zap.Int("jobs", jobsCreated),
	)
	c.JSON(http.StatusCreated, gin.H{
		"account_id":        createdAccountID,
		"account_name":      req.AccountName,
		"exophones_created": exophonesCreated,
		"jobs_created":      jobsCreated,
		"message":           fmt.Sprintf("Organisation '%s' created with %d exophone(s) and %d monitoring jobs", req.AccountName, exophonesCreated, jobsCreated),
	})
}

// AddExophone adds a single virtual number to an existing account and creates its
// 3 monitoring jobs (HEARTBEAT, CALLS, STREAMS) atomically.
// The number is validated against Exotel before being persisted.
// POST /api/v1/accounts/:account_id/exophones
func AddExophone(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	accountID := parseUint(c.Param("account_id"))
	if accountID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	var req struct {
		Number   string `json:"number"   binding:"required"`
		Priority string `json:"priority"` // P0–P3; defaults to P1
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	priority := strings.ToUpper(req.Priority)
	if _, ok := fallbackFrequency[priority]; !ok {
		priority = "P1"
	}

	account, err := repository.GetAccountByID(accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	secretKey := config.App.Crypto.SecretKey
	apiKey, err := utils.Decrypt(account.APIKey, secretKey)
	if err != nil {
		apiKey = account.APIKey
	}
	apiToken, err := utils.Decrypt(account.APIToken, secretKey)
	if err != nil {
		apiToken = account.APIToken
	}

	// Validate the number belongs to this Exotel account by querying it directly.
	// Using a targeted lookup instead of listing all numbers avoids pagination gaps.
	canonical := utils.NormalizePhone(req.Number)
	belongs, err := exotel.ValidateExophoneOnAccount(account.SID, account.Subdomain, apiKey, apiToken, canonical)
	if err != nil {
		logger.Log.Error("add exophone: exotel validation failed",
			zap.String("number", canonical),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("exotel validation failed: %s", err.Error())})
		return
	}
	if !belongs {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":  "number does not belong to this Exotel account",
			"number": canonical,
		})
		return
	}

	// Check for duplicate.
	existing, _, _ := repository.GetExophonesByNumber(canonical)
	for _, ep := range existing {
		if ep.AccountID == accountID {
			c.JSON(http.StatusConflict, gin.H{
				"error":      "exophone already exists for this account",
				"exophone_id": ep.ID,
			})
			return
		}
	}

	freqMap, freqErr := repository.GetPriorityConfigMap()
	if freqErr != nil {
		freqMap = fallbackFrequency
	}
	freq := freqMap[priority]

	var (
		createdExophone models.Exophone
		jobIDs          []uint64
	)

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		ep := models.Exophone{
			AccountID:        accountID,
			ExophoneNumber:   canonical,
			Status:           "active",
			Priority:         priority,
			MonitoringFlag:   1,
			IsCronApplicable: 1,
			CacheTTL:         900,
			MetadataJSON:     "{}",
			CreatedBy:        "API",
			UpdatedBy:        "API",
		}
		if err := tx.Create(&ep).Error; err != nil {
			return fmt.Errorf("exophone insert failed: %w", err)
		}
		createdExophone = ep

		digits := canonical
		for _, jt := range []struct {
			prefix  string
			jobType string
			timeout int
		}{
			{"HB", "HEARTBEAT", 30},
			{"CALLS", "CALLS", 60},
			{"STRM", "STREAMS", 30},
		} {
			job := models.MonitoringJob{
				ExophoneID:      ep.ID,
				AccountID:       accountID,
				JobName:         fmt.Sprintf("%s-%s-%s", jt.prefix, priority, digits),
				JobType:         jt.jobType,
				Priority:        priority,
				FrequencyMinute: freq,
				TimeoutSeconds:  jt.timeout,
				RetryPolicy:     defaultRetryPolicy,
				NextRunAt:       time.Now(),
				LastStatus:      "PENDING",
				IsActive:        1,
				CreatedBy:       "API",
				UpdatedBy:       "API",
			}
			if err := tx.Create(&job).Error; err != nil {
				return fmt.Errorf("job insert failed for %s: %w", jt.jobType, err)
			}
			jobIDs = append(jobIDs, job.ID)
		}
		return nil
	})

	if txErr != nil {
		logger.Log.Error("add exophone: transaction rolled back",
			zap.String("number", canonical),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(txErr),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": txErr.Error()})
		return
	}

	logger.Log.Info("add exophone: success",
		zap.String("number", canonical),
		zap.Uint64("exophone_id", createdExophone.ID),
		zap.Uint64s("job_ids", jobIDs),
	)
	c.JSON(http.StatusCreated, gin.H{
		"exophone_id":     createdExophone.ID,
		"exophone_number": canonical,
		"account_id":      accountID,
		"priority":        priority,
		"frequency_minutes": freq,
		"job_ids":         jobIDs,
		"message":         fmt.Sprintf("Exophone %s added with %d monitoring jobs", canonical, len(jobIDs)),
	})
}

// SyncExophones fetches all active VNs for an account from Exotel and upserts them
// into the DB. New numbers get monitoring jobs; existing ones are left untouched.
// POST /api/v1/accounts/:account_id/exophones/sync
func SyncExophones(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	accountID := parseUint(c.Param("account_id"))
	if accountID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	account, err := repository.GetAccountByID(accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	secretKey := config.App.Crypto.SecretKey
	apiKey, err := utils.Decrypt(account.APIKey, secretKey)
	if err != nil {
		apiKey = account.APIKey
	}
	apiToken, err := utils.Decrypt(account.APIToken, secretKey)
	if err != nil {
		apiToken = account.APIToken
	}

	// Fetch all VNs from Exotel.
	vnSet, err := exotel.ListIncomingPhoneNumbers(account.SID, account.Subdomain, apiKey, apiToken)
	if err != nil {
		logger.Log.Error("sync exophones: exotel fetch failed",
			zap.Uint64("account_id", accountID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("exotel fetch failed: %s", err.Error())})
		return
	}

	// Load existing exophones from DB to detect new vs existing.
	existing, _, _ := repository.GetExophonesByAccount(accountID, 1, 10000)
	existingSet := make(map[string]struct{}, len(existing))
	for _, ep := range existing {
		existingSet[ep.ExophoneNumber] = struct{}{}
	}

	freqMap, freqErr := repository.GetPriorityConfigMap()
	if freqErr != nil {
		freqMap = fallbackFrequency
	}

	var added []string
	for number := range vnSet {
		if _, exists := existingSet[number]; exists {
			continue // already tracked
		}

		// New VN — insert exophone + 3 monitoring jobs.
		ep := models.Exophone{
			AccountID:        accountID,
			ExophoneNumber:   utils.NormalizePhone(number),
			Status:           "active",
			Priority:         "P1",
			MonitoringFlag:   1,
			IsCronApplicable: 1,
			CacheTTL:         900,
			MetadataJSON:     "{}",
			CreatedBy:        "SYNC",
			UpdatedBy:        "SYNC",
		}
		if err := repository.UpsertExophone(&ep); err != nil {
			logger.Log.Warn("sync exophones: failed to insert exophone",
				zap.String("number", number), zap.Error(err))
			continue
		}

		digits := utils.NormalizePhone(number)
		freq := freqMap["P1"]
		for _, jt := range []struct {
			prefix  string
			jobType string
			timeout int
		}{
			{"HB", "HEARTBEAT", 30},
			{"CALLS", "CALLS", 60},
			{"STRM", "STREAMS", 30},
		} {
			job := models.MonitoringJob{
				ExophoneID:      ep.ID,
				AccountID:       accountID,
				JobName:         fmt.Sprintf("%s-P1-%s", jt.prefix, digits),
				JobType:         jt.jobType,
				Priority:        "P1",
				FrequencyMinute: freq,
				TimeoutSeconds:  jt.timeout,
				RetryPolicy:     defaultRetryPolicy,
				NextRunAt:       time.Now(),
				LastStatus:      "PENDING",
				IsActive:        1,
				CreatedBy:       "SYNC",
				UpdatedBy:       "SYNC",
			}
			_ = database.DB.Create(&job).Error
		}
		_ = repository.UpdateLastSynced(ep.ID)
		added = append(added, number)
	}

	logger.Log.Info("sync exophones: complete",
		zap.Uint64("account_id", accountID),
		zap.Int("total_from_exotel", len(vnSet)),
		zap.Int("already_tracked", len(existing)),
		zap.Int("newly_added", len(added)),
	)
	c.JSON(http.StatusOK, gin.H{
		"account_id":      accountID,
		"total_from_exotel": len(vnSet),
		"already_tracked": len(existingSet),
		"newly_added":     len(added),
		"added_numbers":   added,
	})
}

// GetExophone returns the full Exophone record by ID.
// GET /api/v1/exophones/:id
func GetExophone(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	exophoneID := parseUint(c.Param("id"))
	if exophoneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exophone id: must be a positive integer"})
		return
	}

	var e models.Exophone
	if err := database.DB.Where("id = ? AND is_deleted = 0", exophoneID).First(&e).Error; err != nil {
		logger.Log.Error("get exophone: not found",
			zap.Uint64("exophone_id", exophoneID),
			zap.String("request_id", fmt.Sprintf("%v", reqID)),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("exophone %d not found: %s", exophoneID, err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": e})
}
