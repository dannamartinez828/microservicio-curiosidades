package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Microservicio de LECTURA de respaldo (Go). Tiene la misma API que el
// microservicio principal de Node.js (index.js): si Node falla, Django llama
// a este y la app sigue mostrando la informacion desde la misma base de Neon.

const (
	servicio  = "ms-leer-go"
	operacion = "LECTURA (respaldo de Node.js)"
)

var (
	db                *sql.DB
	rutaEspecie       = regexp.MustCompile(`^/api/curiosidades/([^/]+)$`)
	rutaMonedas       = regexp.MustCompile(`^/api/monedas/(\d+)$`)
	rutaMonedasAccion = regexp.MustCompile(`^/api/monedas/(\d+)/(ganar|gastar)$`)
)

// ---------- Conexion a Neon ----------
// Neon agrega parametros (como channel_binding) que el driver no entiende,
// asi que se reconstruye la URL dejando solo lo necesario.
func conectarBD() error {
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		return fmt.Errorf("falta la variable de entorno DATABASE_URL")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	// el driver solo acepta disable, require, verify-ca y verify-full
	sslmode := u.Query().Get("sslmode")
	if sslmode == "" || sslmode == "prefer" || sslmode == "allow" {
		sslmode = "require"
	}
	q := url.Values{}
	q.Set("sslmode", sslmode)
	q.Set("connect_timeout", "10")
	u.RawQuery = q.Encode()
	u.Scheme = "postgres"

	db, err = sql.Open("postgres", u.String())
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxIdleTime(time.Minute) // Neon cierra las conexiones inactivas
	return nil
}

func main() {
	if err := conectarBD(); err != nil {
		log.Fatal(err)
	}
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "7000"
	}
	srv := &http.Server{
		Addr:              ":" + puerto,
		Handler:           http.HandlerFunc(enrutar),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	log.Printf("%s escuchando en el puerto %s", servicio, puerto)
	log.Fatal(srv.ListenAndServe())
}

// ---------- Enrutador ----------
func enrutar(w http.ResponseWriter, r *http.Request) {
	ruta, metodo := r.URL.Path, r.Method
	switch {
	case ruta == "/":
		solo(w, metodo, http.MethodGet, func() { estado(w, r) })
	case ruta == "/openapi.json":
		solo(w, metodo, http.MethodGet, func() { openapi(w, r) })
	case ruta == "/api-docs":
		solo(w, metodo, http.MethodGet, func() { apiDocs(w) })
	case ruta == "/api/curiosidades":
		solo(w, metodo, http.MethodGet, func() { listarCuriosidades(w, r, "") })
	case rutaEspecie.MatchString(ruta):
		especie := rutaEspecie.FindStringSubmatch(ruta)[1]
		solo(w, metodo, http.MethodGet, func() { listarCuriosidades(w, r, especie) })
	case rutaMonedas.MatchString(ruta):
		solo(w, metodo, http.MethodGet, func() { verMonedas(w, r, idMascota(rutaMonedas.FindStringSubmatch(ruta)[1])) })
	case rutaMonedasAccion.MatchString(ruta):
		m := rutaMonedasAccion.FindStringSubmatch(ruta)
		id := idMascota(m[1])
		if m[2] == "ganar" {
			solo(w, metodo, http.MethodPost, func() { ganarMonedas(w, r, id) })
		} else {
			solo(w, metodo, http.MethodPost, func() { gastarMonedas(w, r, id) })
		}
	default:
		enviar(w, http.StatusNotFound, mensajeError("ruta no encontrada"))
	}
}

func solo(w http.ResponseWriter, metodo, esperado string, f func()) {
	if metodo != esperado {
		enviar(w, http.StatusMethodNotAllowed, mensajeError("metodo no permitido"))
		return
	}
	f()
}

// ---------- Estado del servicio (raiz) ----------
type respuestaEstado struct {
	Servicio      string `json:"servicio"`
	Lenguaje      string `json:"lenguaje"`
	Operacion     string `json:"operacion"`
	Estado        string `json:"estado"`
	BaseDeDatos   string `json:"base_de_datos"`
	Documentacion string `json:"documentacion"`
}

func estado(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	ok := db.PingContext(ctx) == nil

	bd, est, codigo := "conectada", "ok", http.StatusOK
	if !ok {
		bd, est, codigo = "error", "error", http.StatusServiceUnavailable
	}
	enviar(w, codigo, respuestaEstado{
		Servicio:      servicio,
		Lenguaje:      "Go",
		Operacion:     operacion,
		Estado:        est,
		BaseDeDatos:   bd,
		Documentacion: "/api-docs",
	})
}

// ---------- Curiosidades (lectura) ----------
type curiosidad struct {
	ID      int    `json:"id"`
	Especie string `json:"especie"`
	Texto   string `json:"texto"`
}

func listarCuriosidades(w http.ResponseWriter, r *http.Request, especie string) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	consulta := "SELECT id, especie, texto FROM curiosidades ORDER BY id"
	args := []any{}
	if especie != "" {
		consulta = "SELECT id, especie, texto FROM curiosidades WHERE especie = $1 ORDER BY id"
		args = append(args, especie)
	}
	filas, err := db.QueryContext(ctx, consulta, args...)
	if err != nil {
		log.Println("error consultando curiosidades:", err)
		enviar(w, http.StatusInternalServerError, mensajeError("Error consultando la base de datos"))
		return
	}
	defer filas.Close()

	lista := []curiosidad{} // [] y no null cuando no hay resultados
	for filas.Next() {
		var c curiosidad
		if err := filas.Scan(&c.ID, &c.Especie, &c.Texto); err != nil {
			log.Println("error leyendo fila:", err)
			enviar(w, http.StatusInternalServerError, mensajeError("Error consultando la base de datos"))
			return
		}
		lista = append(lista, c)
	}
	enviar(w, http.StatusOK, lista)
}

// ---------- Monedas ----------
type saldo struct {
	MascotaID int64 `json:"mascota_id"`
	Cantidad  int64 `json:"cantidad"`
}

type resultadoGasto struct {
	Ok        bool  `json:"ok"`
	MascotaID int64 `json:"mascota_id"`
	Cantidad  int64 `json:"cantidad"`
}

func idMascota(s string) int64 {
	id, err := strconv.ParseInt(s, 10, 32) // la columna es INTEGER
	if err != nil {
		return -1
	}
	return id
}

func verMonedas(w http.ResponseWriter, r *http.Request, id int64) {
	if id < 0 {
		enviar(w, http.StatusNotFound, mensajeError("mascota no encontrada"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	var cantidad int64
	err := db.QueryRowContext(ctx, "SELECT cantidad FROM monedas WHERE mascota_id = $1", id).Scan(&cantidad)
	if err != nil && err != sql.ErrNoRows { // sin fila = nunca ha jugado = 0 monedas
		log.Println("error consultando monedas:", err)
		enviar(w, http.StatusInternalServerError, mensajeError("Error consultando la base de datos"))
		return
	}
	enviar(w, http.StatusOK, saldo{MascotaID: id, Cantidad: cantidad})
}

// cantidad del cuerpo: acepta numero o texto numerico (igual que el de Node)
func cantidadDe(r *http.Request) (int64, bool) {
	cuerpo := leerJSON(r)
	var n float64
	switch v := cuerpo["cantidad"].(type) {
	case float64:
		n = v
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			f = 0
		}
		n = f
	}
	if n < 0 {
		return 0, false
	}
	return int64(n), true
}

func ganarMonedas(w http.ResponseWriter, r *http.Request, id int64) {
	cantidad, ok := cantidadDe(r)
	if id < 0 || !ok {
		enviar(w, http.StatusBadRequest, mensajeError("cantidad invalida"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	var nuevo int64
	// una sola sentencia atomica: crea la fila si no existe o suma al saldo
	err := db.QueryRowContext(ctx,
		"INSERT INTO monedas (mascota_id, cantidad) VALUES ($1, $2) "+
			"ON CONFLICT (mascota_id) DO UPDATE SET cantidad = monedas.cantidad + EXCLUDED.cantidad "+
			"RETURNING cantidad",
		id, cantidad,
	).Scan(&nuevo)
	if err != nil {
		log.Println("error sumando monedas:", err)
		enviar(w, http.StatusInternalServerError, mensajeError("Error actualizando la base de datos"))
		return
	}
	enviar(w, http.StatusOK, saldo{MascotaID: id, Cantidad: nuevo})
}

func gastarMonedas(w http.ResponseWriter, r *http.Request, id int64) {
	cantidad, ok := cantidadDe(r)
	if id < 0 || !ok {
		enviar(w, http.StatusBadRequest, mensajeError("cantidad invalida"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx,
		"INSERT INTO monedas (mascota_id, cantidad) VALUES ($1, 0) ON CONFLICT (mascota_id) DO NOTHING", id); err != nil {
		log.Println("error preparando saldo:", err)
		enviar(w, http.StatusInternalServerError, mensajeError("Error actualizando la base de datos"))
		return
	}

	// resta solo si alcanza (atomico: dos compras a la vez no dejan el saldo negativo)
	var nuevo int64
	err := db.QueryRowContext(ctx,
		"UPDATE monedas SET cantidad = cantidad - $2 WHERE mascota_id = $1 AND cantidad >= $2 RETURNING cantidad",
		id, cantidad,
	).Scan(&nuevo)
	if err == sql.ErrNoRows {
		var actual int64
		if err := db.QueryRowContext(ctx, "SELECT cantidad FROM monedas WHERE mascota_id = $1", id).Scan(&actual); err != nil {
			log.Println("error leyendo saldo:", err)
			enviar(w, http.StatusInternalServerError, mensajeError("Error consultando la base de datos"))
			return
		}
		enviar(w, http.StatusOK, resultadoGasto{Ok: false, MascotaID: id, Cantidad: actual})
		return
	}
	if err != nil {
		log.Println("error restando monedas:", err)
		enviar(w, http.StatusInternalServerError, mensajeError("Error actualizando la base de datos"))
		return
	}
	enviar(w, http.StatusOK, resultadoGasto{Ok: true, MascotaID: id, Cantidad: nuevo})
}

// ---------- Documentacion Swagger (/api-docs) ----------
const especificacion = `{
  "openapi": "3.0.0",
  "info": {
    "title": "API - Microservicio de LECTURA de respaldo (Go)",
    "version": "1.0.0",
    "description": "Misma API que el microservicio de Node.js. Si Node falla, Django usa este para seguir mostrando curiosidades y monedas desde Neon."
  },
  "servers": [{"url": "%s"}],
  "paths": {
    "/": {"get": {"summary": "Estado del servicio", "responses": {"200": {"description": "Servicio y base de datos funcionando"}, "503": {"description": "Sin conexion con la base de datos"}}}},
    "/api/curiosidades": {"get": {"summary": "Todas las curiosidades", "responses": {"200": {"description": "Lista completa de curiosidades"}, "500": {"description": "Error consultando la base de datos"}}}},
    "/api/curiosidades/{especie}": {"get": {"summary": "Curiosidades filtradas por especie", "parameters": [{"in": "path", "name": "especie", "required": true, "schema": {"type": "string", "example": "perro"}}], "responses": {"200": {"description": "Lista de curiosidades de esa especie"}, "500": {"description": "Error consultando la base de datos"}}}},
    "/api/monedas/{mascota_id}": {"get": {"summary": "Saldo de monedas de una mascota", "parameters": [{"in": "path", "name": "mascota_id", "required": true, "schema": {"type": "integer", "example": 1}}], "responses": {"200": {"description": "Saldo actual (0 si nunca ha jugado)"}}}},
    "/api/monedas/{mascota_id}/ganar": {"post": {"summary": "Sumar monedas", "parameters": [{"in": "path", "name": "mascota_id", "required": true, "schema": {"type": "integer", "example": 1}}], "requestBody": {"required": true, "content": {"application/json": {"schema": {"type": "object", "properties": {"cantidad": {"type": "integer", "example": 10}}}}}}, "responses": {"200": {"description": "Nuevo saldo"}, "400": {"description": "Cantidad invalida"}}}},
    "/api/monedas/{mascota_id}/gastar": {"post": {"summary": "Gastar monedas (solo si alcanzan)", "parameters": [{"in": "path", "name": "mascota_id", "required": true, "schema": {"type": "integer", "example": 1}}], "requestBody": {"required": true, "content": {"application/json": {"schema": {"type": "object", "properties": {"cantidad": {"type": "integer", "example": 10}}}}}}, "responses": {"200": {"description": "ok=true si alcanzaba; ok=false y el saldo actual si no"}, "400": {"description": "Cantidad invalida"}}}}
  }
}`

func openapi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, especificacion, servidorPublico(r))
}

func apiDocs(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, `<!doctype html>
<html><head><meta charset="utf-8"><title>API - Leer (Go, respaldo)</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({ url: "/openapi.json", dom_id: "#swagger-ui" });</script>
</body></html>`)
}

// URL publica del servicio (respeta https detras del proxy de Render)
func servidorPublico(r *http.Request) string {
	proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	if proto == "" {
		proto = "http"
	}
	host := r.Host
	if host == "" {
		host = "localhost"
	}
	return proto + "://" + host
}

// ---------- Utilidades ----------
func leerJSON(r *http.Request) map[string]any {
	cuerpo := map[string]any{}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil || json.Unmarshal(data, &cuerpo) != nil {
		return map[string]any{}
	}
	return cuerpo
}

type errorJSON struct {
	Error string `json:"error"`
}

func mensajeError(m string) errorJSON { return errorJSON{Error: m} }

func enviar(w http.ResponseWriter, codigo int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(cuerpo)
}
