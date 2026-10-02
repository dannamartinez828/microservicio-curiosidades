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
	servicio  = "ms-eliminar-go"
	operacion = "ELIMINACION"
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
		puerto = "7003"
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
	case rutaID.MatchString(ruta):
		if r.Method == http.MethodDelete {
			eliminar(w, r, idDe(ruta))
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

// ---------- Eliminar ----------
func eliminar(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	var eliminado int
	err := db.QueryRowContext(ctx, "DELETE FROM curiosidades WHERE id = $1 RETURNING id", id).Scan(&eliminado)
	if err == sql.ErrNoRows {
		enviar(w, http.StatusNotFound, mensajeError("curiosidad no encontrada"))
		return
	}
	if err != nil {
		log.Println("error eliminando:", err)
		enviar(w, http.StatusInternalServerError, mensajeError("Error escribiendo en la base de datos"))
		return
	}
	enviar(w, http.StatusOK, map[string]int{"eliminado": eliminado})
}

const especificacion = `{
  "openapi": "3.0.0",
  "info": {"title": "API - Microservicio de ELIMINACION (Go)", "version": "1.0.0", "description": "Elimina curiosidades de mascotas de la tabla curiosidades de Neon."},
  "servers": [{"url": "%s"}],
  "paths": {
    "/": {"get": {"summary": "Estado del servicio", "responses": {"200": {"description": "Servicio y base de datos funcionando"}, "503": {"description": "Sin conexion con la base de datos"}}}},
    "/api/curiosidades/{id}": {"delete": {"summary": "Eliminar una curiosidad", "parameters": [{"in": "path", "name": "id", "required": true, "schema": {"type": "integer", "example": 1}}], "responses": {"200": {"description": "Curiosidad eliminada (devuelve el id eliminado)"}, "404": {"description": "Curiosidad no encontrada"}, "500": {"description": "Error escribiendo en la base de datos"}}}}
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
<html><head><meta charset="utf-8"><title>API - Eliminar (Go)</title>
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
