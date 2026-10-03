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

const (
	servicio  = "ms-consultar-go"
	operacion = "LECTURA (respaldo)"
)

var (
	db       *sql.DB
	especies = map[string]bool{"perro": true, "gato": true, "dragon": true, "robot": true}
	rutaID   = regexp.MustCompile(`^/api/curiosidades/(\d+)$`)
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
	sslmode := u.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "prefer"
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
		puerto = "7004"
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
	ruta := r.URL.Path
	switch {
	case ruta == "/":
		if r.Method == http.MethodGet {
			estado(w, r)
		} else {
			noPermitido(w)
		}
	case ruta == "/openapi.json":
		if r.Method == http.MethodGet {
			openapi(w, r)
		} else {
			noPermitido(w)
		}
	case ruta == "/api-docs":
		if r.Method == http.MethodGet {
			apiDocs(w)
		} else {
			noPermitido(w)
		}
	case ruta == "/api/curiosidades":
		if r.Method == http.MethodGet {
			listarCuriosidades(w, r, "")
		} else {
			noPermitido(w)
		}
	case strings.HasPrefix(ruta, "/api/curiosidades/") && strings.TrimPrefix(ruta, "/api/curiosidades/") != "" &&
		!strings.Contains(strings.TrimPrefix(ruta, "/api/curiosidades/"), "/"):
		if r.Method == http.MethodGet {
			listarCuriosidades(w, r, strings.TrimPrefix(ruta, "/api/curiosidades/"))
		} else {
			noPermitido(w)
		}
	case rutaMonedas.MatchString(ruta):
		if r.Method == http.MethodGet {
			saldoMonedas(w, r, idMonedas(ruta))
		} else {
			noPermitido(w)
		}
	default:
		enviar(w, http.StatusNotFound, mensajeError("ruta no encontrada"))
	}
}

// ---------- Estado del servicio (raiz) ----------
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

type respuestaEstado struct {
	Servicio      string `json:"servicio"`
	Lenguaje      string `json:"lenguaje"`
	Operacion     string `json:"operacion"`
	Estado        string `json:"estado"`
	BaseDeDatos   string `json:"base_de_datos"`
	Documentacion string `json:"documentacion"`
}

type curiosidad struct {
	ID      int    `json:"id"`
	Especie string `json:"especie"`
	Texto   string `json:"texto"`
}

// ---------- Consultas (solo lectura): mismo contrato que el microservicio Node ----------
var rutaMonedas = regexp.MustCompile(`^/api/monedas/(\d+)$`)

type saldo struct {
	MascotaID int64 `json:"mascota_id"`
	Cantidad  int   `json:"cantidad"`
}

func idMonedas(ruta string) int64 {
	m := rutaMonedas.FindStringSubmatch(ruta)
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return -1
	}
	return id
}

// GET /api/curiosidades y GET /api/curiosidades/{especie}
func listarCuriosidades(w http.ResponseWriter, r *http.Request, especie string) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var filas *sql.Rows
	var err error
	if especie == "" {
		filas, err = db.QueryContext(ctx, "SELECT id, especie, texto FROM curiosidades ORDER BY id")
	} else {
		filas, err = db.QueryContext(ctx, "SELECT id, especie, texto FROM curiosidades WHERE especie = $1 ORDER BY id", especie)
	}
	if err != nil {
		log.Println("error consultando curiosidades:", err)
		enviar(w, http.StatusInternalServerError, mensajeError("Error consultando la base de datos"))
		return
	}
	defer filas.Close()

	lista := []curiosidad{}
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

// GET /api/monedas/{mascota_id}: si la mascota nunca ha jugado, el saldo es 0
func saldoMonedas(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	var cantidad int
	err := db.QueryRowContext(ctx, "SELECT cantidad FROM monedas WHERE mascota_id = $1", id).Scan(&cantidad)
	if err != nil && err != sql.ErrNoRows {
		log.Println("error consultando monedas:", err)
		enviar(w, http.StatusInternalServerError, mensajeError("Error consultando la base de datos"))
		return
	}
	enviar(w, http.StatusOK, saldo{MascotaID: id, Cantidad: cantidad})
}

const especificacion = `{
  "openapi": "3.0.0",
  "info": {"title": "API - Microservicio de CONSULTA de respaldo (Go)", "version": "1.0.0", "description": "Servicio de lectura de respaldo: si el microservicio principal en Node.js falla, la app Django consulta este para seguir mostrando curiosidades y monedas."},
  "servers": [{"url": "%s"}],
  "paths": {
    "/": {"get": {"summary": "Estado del servicio", "responses": {"200": {"description": "Servicio y base de datos funcionando"}, "503": {"description": "Sin conexion con la base de datos"}}}},
    "/api/curiosidades": {"get": {"summary": "Todas las curiosidades", "responses": {"200": {"description": "Lista de curiosidades"}, "500": {"description": "Error consultando la base de datos"}}}},
    "/api/curiosidades/{especie}": {"get": {"summary": "Curiosidades de una especie", "parameters": [{"in": "path", "name": "especie", "required": true, "schema": {"type": "string", "example": "perro"}}], "responses": {"200": {"description": "Lista de curiosidades de esa especie"}, "500": {"description": "Error consultando la base de datos"}}}},
    "/api/monedas/{mascota_id}": {"get": {"summary": "Saldo de monedas de una mascota", "parameters": [{"in": "path", "name": "mascota_id", "required": true, "schema": {"type": "integer", "example": 1}}], "responses": {"200": {"description": "Saldo actual (0 si nunca ha jugado)"}, "500": {"description": "Error consultando la base de datos"}}}}
  }
}`

// ---------- Documentacion Swagger (/api-docs) ----------
func openapi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, especificacion, servidorPublico(r))
}

func apiDocs(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, `<!doctype html>
<html><head><meta charset="utf-8"><title>API - Consultar (Go, respaldo)</title>
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

func campo(cuerpo map[string]any, nombre string) string {
	if s, ok := cuerpo[nombre].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func idDe(ruta string) int64 {
	m := rutaID.FindStringSubmatch(ruta)
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return -1
	}
	return id
}

type errorJSON struct {
	Error string `json:"error"`
}

func mensajeError(m string) errorJSON { return errorJSON{Error: m} }

func noPermitido(w http.ResponseWriter) {
	enviar(w, http.StatusMethodNotAllowed, mensajeError("metodo no permitido"))
}

func enviar(w http.ResponseWriter, codigo int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(cuerpo)
}
