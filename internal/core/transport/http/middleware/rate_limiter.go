package core_http_middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	core_errors "github.com/rubtsov-ilya/golang-starter/internal/core/errors"
	core_logger "github.com/rubtsov-ilya/golang-starter/internal/core/logger"
	core_http_response "github.com/rubtsov-ilya/golang-starter/internal/core/transport/http/response"
)

// client хранит лимитер для конкретного IP и время его последнего запроса.
type client struct {
	limiter  *rate.Limiter
	mu       sync.Mutex // защищает поле lastSeen при конкурентных запросах от одного IP
	lastSeen time.Time
}

// IPRateLimiter управляет лимитерами для всех IP-адресов.
type IPRateLimiter struct {
	ips             map[string]*client
	mu              sync.RWMutex // защищает карту ips
	rps             rate.Limit   // количество токенов в секунду (RPS)
	burst           int          // максимальный всплеск (burst)
	cleanupInterval time.Duration
	lifetime        time.Duration
	stopCleanup     chan struct{}
	wg              sync.WaitGroup
}

// NewIPRateLimiter создает и запускает новый IPRateLimiter.
//   - rps: лимит запросов в секунду (RPS)
//   - burst: максимальный всплеск (burst)
//   - cleanupInterval: как часто запускать очистку неактивных клиентов в фоне
//   - lifetime: сколько времени клиент считается активным после последнего запроса
func NewIPRateLimiter(rps float64, burst int, cleanupInterval, lifetime time.Duration) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips:             make(map[string]*client),
		rps:             rate.Limit(rps),
		burst:           burst,
		cleanupInterval: cleanupInterval,
		lifetime:        lifetime,
		stopCleanup:     make(chan struct{}),
	}

	limiter.wg.Add(1)
	go limiter.cleanupLoop()

	return limiter
}

// Close останавливает фоновый процесс очистки памяти.
func (l *IPRateLimiter) Close() {
	close(l.stopCleanup)
	l.wg.Wait()
}

// cleanupLoop — фоновая горутина, которая периодически очищает неактивные IP
// для предотвращения утечки памяти.
func (l *IPRateLimiter) cleanupLoop() {
	defer l.wg.Done()
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			now := time.Now()
			for ip, c := range l.ips {
				c.mu.Lock()
				idleTime := now.Sub(c.lastSeen)
				c.mu.Unlock()

				if idleTime > l.lifetime {
					delete(l.ips, ip)
				}
			}
			l.mu.Unlock()
		case <-l.stopCleanup:
			return
		}
	}
}

// getLimiter возвращает существующий лимитер для IP или создает новый.
func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	// Сначала проверяем наличие под быстрой блокировкой чтения (RLock)
	l.mu.RLock()
	c, exists := l.ips[ip]
	l.mu.RUnlock()

	if !exists {
		// Если лимитера нет, берем блокировку на запись и создаем его
		l.mu.Lock()
		// Двойная проверка (Double-check), так как другая горутина
		// могла успеть создать клиента, пока мы переключали блокировку
		c, exists = l.ips[ip]
		if !exists {
			c = &client{
				limiter:  rate.NewLimiter(l.rps, l.burst),
				lastSeen: time.Now(),
			}
			l.ips[ip] = c
		}
		l.mu.Unlock()
	} else {
		// Если клиент уже есть, обновляем время активности.
		// Используем mutex внутри клиента c.mu, чтобы не блокировать всю карту l.ips
		c.mu.Lock()
		c.lastSeen = time.Now()
		c.mu.Unlock()
	}

	return c.limiter
}

// RateLimiter возвращает Middleware, реализующий лимитирование запросов.
func (l *IPRateLimiter) RateLimiter() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := l.getClientIP(r)
			limiter := l.getLimiter(ip)

			// Резервируем токен. Reserve() всегда успешно возвращает Reservation,
			// за исключением случаев, когда Burst равен 0.
			res := limiter.Reserve()
			if !res.OK() {
				// Всплеск (Burst) равен 0, отклоняем сразу
				l.reject(w, r, 1)
				return
			}

			// Если Delay > 0, значит лимит превышен и нужно подождать
			delay := res.Delay()
			if delay > 0 {
				// Отменяем резервацию, так как отклоняется запрос (HTTP 429)
				// и не нцжно тратить токен на отклоненного клиента
				res.Cancel()

				// Округляем время ожидания в большую сторону до секунд
				retryAfterSecs := int(math.Ceil(delay.Seconds()))
				if retryAfterSecs < 1 {
					retryAfterSecs = 1
				}

				l.reject(w, r, retryAfterSecs)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// reject логирует блокировку, выставляет заголовок Retry-After и возвращает JSON-ответ HTTP 429.
func (l *IPRateLimiter) reject(w http.ResponseWriter, r *http.Request, retryAfter int) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))

	ctx := r.Context()
	log := core_logger.FromContext(ctx)

	// Логируем предупреждение о превышении лимита
	log.Warn("rate limit exceeded",
		core_logger.String("ip", l.getClientIP(r)),
		core_logger.Int("retry_after_seconds", retryAfter),
	)

	responseHandler := core_http_response.NewHTTPResponseHandler(log, w)
	responseHandler.ErrorResponse(core_errors.ErrTooManyRequests, "too many requests, please try again later")
}

// getClientIP вытаскивает реальный IP-адрес клиента с учетом прокси-серверов.
func (l *IPRateLimiter) getClientIP(r *http.Request) string {
	// Заголовок X-Forwarded-For может содержать цепочку IP: "client, proxy1, proxy2"
	// Первый IP в списке является реальным IP клиента
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}

	// Заголовок X-Real-IP обычно проставляется Nginx
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	// Если заголовков нет, берем RemoteAddr (обычно "IP:port")
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
