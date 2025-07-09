package main

import (
	"context"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var (
	redisAddr     string
	redisPassword string
	redisDB       int
	matchPattern  string
	workerCount   int
	shortTTL      int
	exportPath    string
	useTLS        bool
)

var ctx = context.Background()

type TTLStats struct {
	NoExpiry     int
	Expired      int
	Short        int
	Long         int
	SizeNoExpiry int64
	SizeExpired  int64
	SizeShort    int64
	SizeLong     int64
	sync.Mutex
}

type DupStats struct {
	Count int
	Keys  []string
	Size  int
	TTLs  map[string]time.Duration
	sync.Mutex
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "redis-analyzer",
		Short: "Redis analyzer for TTLs and duplicate values",
		Run:   runAnalyzer,
	}

	rootCmd.Flags().StringVar(&redisAddr, "addr", "localhost:6379", "Redis server address")
	rootCmd.Flags().StringVar(&redisPassword, "password", "", "Redis password")
	rootCmd.Flags().IntVar(&redisDB, "db", 0, "Redis database")
	rootCmd.Flags().StringVar(&matchPattern, "match", "*", "Key pattern to match")
	rootCmd.Flags().IntVar(&workerCount, "workers", 5, "Number of worker goroutines")
	rootCmd.Flags().IntVar(&shortTTL, "short-ttl", 3600, "Threshold (in seconds) for short TTL")
	rootCmd.Flags().StringVar(&exportPath, "export", "", "Path to CSV file for export (optional)")
	rootCmd.Flags().BoolVar(&useTLS, "tls", false, "Enable TLS connection to Redis")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runAnalyzer(cmd *cobra.Command, args []string) {
	opts := &redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	}
	if useTLS {
		opts.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	rdb := redis.NewClient(opts)

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", redisAddr, err)
	}
	fmt.Printf("✅ Connected to Redis at %s\n", redisAddr)

	keyChan := make(chan string, 1000)
	wg := sync.WaitGroup{}

	ttlStats := TTLStats{}
	hashMap := sync.Map{}

	var processedKeys int64 = 0

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range keyChan {
				ttl, err := rdb.TTL(ctx, key).Result()

				val, errVal := rdb.Get(ctx, key).Bytes()

				if err == nil && errVal == nil {
					ttlStats.Lock()
					size := int64(len(val))
					switch {
					case ttl < 0:
						if ttl == -1 {
							ttlStats.NoExpiry++
							ttlStats.SizeNoExpiry += size
						} else {
							ttlStats.Expired++
							ttlStats.SizeExpired += size
						}
					case ttl < time.Duration(shortTTL)*time.Second:
						ttlStats.Short++
						ttlStats.SizeShort += size
					default:
						ttlStats.Long++
						ttlStats.SizeLong += size
					}
					ttlStats.Unlock()

					hash := fmt.Sprintf("%x", xxhash.Sum64(val))
					actual, _ := hashMap.LoadOrStore(hash, &DupStats{Count: 0, Keys: []string{}, Size: 0, TTLs: make(map[string]time.Duration)})
					stats := actual.(*DupStats)
					stats.Lock()
					stats.Count++
					stats.Keys = append(stats.Keys, key)
					stats.Size += len(val)
					stats.TTLs[key] = ttl
					stats.Unlock()
				}

				atomic.AddInt64(&processedKeys, 1)
				if processedKeys%1000 == 0 {
					fmt.Printf("⏳ Processed keys: %d\n", processedKeys)
				}
			}
		}()
	}

	fmt.Printf("🔍 Starting SCAN with pattern '%s'...\n", matchPattern)

	cursor := uint64(0)
	for {
		keys, next, err := rdb.Scan(ctx, cursor, matchPattern, 500).Result()
		if err != nil {
			log.Fatalf("SCAN error: %v", err)
		}
		for _, key := range keys {
			keyChan <- key
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	close(keyChan)
	wg.Wait()

	fmt.Printf("✅ Scan complete. Total keys processed: %d\n\n", processedKeys)

	fmt.Println("TTL Stats:")
	fmt.Printf("  No Expiry: %d (%.2f MB)\n", ttlStats.NoExpiry, float64(ttlStats.SizeNoExpiry)/1024/1024)
	fmt.Printf("  Expired:   %d (%.2f MB)\n", ttlStats.Expired, float64(ttlStats.SizeExpired)/1024/1024)
	fmt.Printf("  Short TTL: %d (%.2f MB)\n", ttlStats.Short, float64(ttlStats.SizeShort)/1024/1024)
	fmt.Printf("  Long TTL:  %d (%.2f MB)\n", ttlStats.Long, float64(ttlStats.SizeLong)/1024/1024)
	fmt.Printf("  Total Size: %.2f MB\n",
		float64(ttlStats.SizeNoExpiry+ttlStats.SizeExpired+ttlStats.SizeShort+ttlStats.SizeLong)/1024/1024)

	fmt.Println("\nDuplicate values:")
	if exportPath != "" {
		exportToCSV(&hashMap, exportPath)
	} else {
		hashMap.Range(func(_, v interface{}) bool {
			stats := v.(*DupStats)
			if stats.Count > 1 {
				fmt.Printf("  Count: %d, Size: %.2f KB, Sample: %s\n", stats.Count, float64(stats.Size)/1024.0, freshestKey(stats.TTLs))
			}
			return true
		})
	}
}

func freshestKey(ttls map[string]time.Duration) string {
	var freshest string
	var maxTTL time.Duration = -2
	for k, ttl := range ttls {
		if ttl > maxTTL {
			freshest = k
			maxTTL = ttl
		}
	}
	return freshest
}

func exportToCSV(hashMap *sync.Map, path string) {
	type row struct {
		Count  int
		SizeKB float64
		Sample string
	}

	var rows []row

	hashMap.Range(func(_, v interface{}) bool {
		stats := v.(*DupStats)
		if stats.Count > 1 {
			rows = append(rows, row{
				Count:  stats.Count,
				SizeKB: float64(stats.Size) / 1024.0,
				Sample: freshestKey(stats.TTLs),
			})
		}
		return true
	})

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].SizeKB > rows[j].SizeKB
	})

	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("CSV export failed: %v", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"count", "size_kb", "sample"})

	for _, r := range rows {
		w.Write([]string{
			strconv.Itoa(r.Count),
			fmt.Sprintf("%.2f", r.SizeKB),
			r.Sample,
		})
	}

	fmt.Printf("CSV export written to: %s\n", path)
}
