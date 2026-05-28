// Package service 提供证书管理、Nginx 管理、自动扫描等核心业务逻辑
package service

import (
	"sync"
	"time"
)

const (
	maxFailedAttempts = 5              // 最大允许失败尝试次数，超过即触发锁定
	lockDuration      = 30 * time.Minute // 锁定时长（30 分钟），到期自动解锁
)

// ipRecord 单个 IP 地址的登录尝试记录
type ipRecord struct {
	Attempts  int       // 累计失败尝试次数
	LockedAt  time.Time // 锁定触发时间（用于计算剩余锁定时长）
	IsLocked  bool      // 当前是否处于锁定状态
}

// IPLockout IP 登录限流锁定服务
// 基于内存的滑动窗口限流器，防止暴力破解登录密码
// 使用 sync.RWMutex 保证并发安全
type IPLockout struct {
	mu      sync.RWMutex          // 读写锁，保护 records 的并发访问
	records map[string]*ipRecord  // IP 地址到登录记录的映射表
}

// NewIPLockout 创建 IP 限流锁定实例
// 初始化记录表并启动后台清理协程，自动清除过期的锁定记录
func NewIPLockout() *IPLockout {
	l := &IPLockout{
		records: make(map[string]*ipRecord),
	}
	go l.cleanup() // 启动后台定期清理过期记录
	return l
}

// Check 检查指定 IP 是否被锁定
// 返回值：locked 表示是否已锁定，remaining 表示剩余允许失败次数
// 逻辑：
//   - IP 无记录 -> 未锁定，剩余 maxFailedAttempts 次
//   - IP 已锁定且未到期 -> 已锁定，剩余 0 次
//   - IP 已锁定但已过期 -> 解锁（视同新 IP）
//   - IP 未锁定 -> 未锁定，返回剩余尝试次数
func (l *IPLockout) Check(ip string) (locked bool, remaining int) {
	l.mu.RLock()         // 获取读锁（允许多个并发读取）
	defer l.mu.RUnlock()

	rec, ok := l.records[ip]
	if !ok {
		return false, maxFailedAttempts // IP 无记录，允许全部尝试次数
	}

	if rec.IsLocked {
		if time.Since(rec.LockedAt) >= lockDuration {
			return false, maxFailedAttempts // 锁定已过期，自动解锁
		}
		return true, 0 // 仍在锁定期内
	}

	return false, maxFailedAttempts - rec.Attempts // 未锁定，返回剩余次数
}

// RecordFailure 记录一次登录失败
// 累加失败计数，达到 maxFailedAttempts 时触发锁定
// 返回值：true 表示已触发锁定（当前或此前已锁定）
func (l *IPLockout) RecordFailure(ip string) (locked bool) {
	l.mu.Lock()         // 获取写锁（修改记录需要独占访问）
	defer l.mu.Unlock()

	rec, ok := l.records[ip]
	if !ok {
		rec = &ipRecord{}         // 首次失败，创建新记录
		l.records[ip] = rec
	}

	if rec.IsLocked {
		return true // 已处于锁定状态
	}

	rec.Attempts++ // 累加失败次数
	if rec.Attempts >= maxFailedAttempts {
		rec.IsLocked = true
		rec.LockedAt = time.Now() // 记录锁定时间
		return true               // 刚刚触发锁定
	}
	return false // 尚未达到锁定阈值
}

// Reset 清除指定 IP 的登录失败记录
// 在登录成功后调用，重置该 IP 的失败计数和锁定状态
func (l *IPLockout) Reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.records, ip) // 直接删除记录
}

// GetLockRemaining 获取指定 IP 的剩余锁定时长
// 返回值：距离自动解锁的剩余时间，若未锁定则返回 0
func (l *IPLockout) GetLockRemaining(ip string) time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()

	rec, ok := l.records[ip]
	if !ok || !rec.IsLocked {
		return 0 // IP 无记录或未锁定
	}

	elapsed := time.Since(rec.LockedAt) // 已过去的时间
	if elapsed >= lockDuration {
		return 0 // 已超过锁定时长，视为已解锁
	}
	return lockDuration - elapsed // 返回剩余锁定时长
}

// cleanup 后台清理协程，定期清除过期的 IP 记录
// 每 10 分钟运行一次，删除已超过锁定时长的记录以释放内存
// 包括：已锁定但过期的记录、未锁定但记录时间过久的记录
func (l *IPLockout) cleanup() {
	ticker := time.NewTicker(10 * time.Minute) // 每 10 分钟清理一次
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock() // 清理需要写锁
		for ip, rec := range l.records {
			if rec.IsLocked && time.Since(rec.LockedAt) >= lockDuration {
				delete(l.records, ip) // 已锁定且过期，清除
			} else if !rec.IsLocked && time.Since(rec.LockedAt) >= lockDuration {
				delete(l.records, ip) // 未锁定但记录时间过久，清除
			}
		}
		l.mu.Unlock()
	}
}
