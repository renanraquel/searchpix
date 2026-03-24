// Package imagecache mantém cache em memória de imagens de produto já redimensionadas.
//
// Query em GET /api/products/image e /api/public/product-image:
//   - sem parâmetro: maior lado = DefaultMaxSide (480px), JPEG Q85;
//   - ?w=0: bytes originais do banco (sem resize);
//   - ?w=200 (64–2048): maior lado em pixels.
//
// Invalidação: chamar InvalidateProduct(id) após create/update/delete do produto.
package imagecache

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// DefaultMaxSide é o maior lado (px) quando o cliente não envia ?w= — bom para miniaturas em listas.
const DefaultMaxSide = 480

// entry guarda bytes já processados (original ou redimensionado + JPEG).
type entry struct {
	data []byte
	ct   string
}

// Cache é um cache em memória por (productID, maxSide). Inicializado vazio no start da aplicação.
type Cache struct {
	mu sync.RWMutex
	m  map[string]entry
}

// New cria o cache vazio (preenchido sob demanda nos GET de imagem).
func New() *Cache {
	return &Cache{m: make(map[string]entry)}
}

func cacheKey(productID string, maxSide int) string {
	return productID + "::" + strconv.Itoa(maxSide)
}

// ParseMaxSide lê ?w= da query: 0 = original do banco; omitido = DefaultMaxSide; limitado a [64, 2048].
func ParseMaxSide(q url.Values) int {
	s := strings.TrimSpace(q.Get("w"))
	if s == "" {
		return DefaultMaxSide
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return DefaultMaxSide
	}
	if n == 0 {
		return 0
	}
	if n < 64 {
		return 64
	}
	if n > 2048 {
		return 2048
	}
	return n
}

// Get busca no cache; ok=false se miss.
func (c *Cache) Get(productID string, maxSide int) (data []byte, contentType string, ok bool) {
	key := cacheKey(productID, maxSide)
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.m[key]
	if !ok {
		return nil, "", false
	}
	return e.data, e.ct, true
}

// Set grava no cache (após fetch + resize).
func (c *Cache) Set(productID string, maxSide int, data []byte, contentType string) {
	if len(data) == 0 {
		return
	}
	key := cacheKey(productID, maxSide)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = entry{data: data, ct: contentType}
}

// GetOrRender: cache hit retorna direto; senão chama fetch (DB), aplica resize se maxSide>0, grava e retorna.
func (c *Cache) GetOrRender(productID string, maxSide int, fetch func() ([]byte, string, error)) ([]byte, string, error) {
	if data, ct, ok := c.Get(productID, maxSide); ok {
		return data, ct, nil
	}
	data, ct, err := fetch()
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", nil
	}
	out, outCT := data, ct
	if maxSide > 0 {
		r, rct, e2 := resizeToMaxJPEG(data, maxSide)
		switch {
		case e2 == nil && len(r) > 0:
			out, outCT = r, rct
		case errors.Is(e2, errSkipResize):
			// mantém original
		default:
			// formato não decodificável etc. — mantém original
		}
	}
	c.Set(productID, maxSide, out, outCT)
	return out, outCT, nil
}

// InvalidateProduct remove todas as entradas daquele produto (qualquer maxSide).
func (c *Cache) InvalidateProduct(productID string) {
	if productID == "" {
		return
	}
	prefix := productID + "::"
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.m {
		if strings.HasPrefix(k, prefix) {
			delete(c.m, k)
		}
	}
}
