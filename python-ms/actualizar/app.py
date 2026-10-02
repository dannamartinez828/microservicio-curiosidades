import os
from contextlib import closing

import psycopg2
from flask import Flask, jsonify, request
from werkzeug.middleware.proxy_fix import ProxyFix

app = Flask(__name__)
app.wsgi_app = ProxyFix(app.wsgi_app, x_proto=1, x_host=1)  # Render: respeta https
app.json.ensure_ascii = False
app.json.sort_keys = False

DATABASE_URL = os.environ["DATABASE_URL"]
ESPECIES = {"perro", "gato", "dragon", "robot"}


def ejecutar(sql, params=()):
    with closing(psycopg2.connect(DATABASE_URL, connect_timeout=10)) as conn:
        with conn, conn.cursor() as cur:
            cur.execute(sql, params)
            return cur.fetchone()


@app.get("/")
def estado():
    try:
        ejecutar("SELECT 1")
        bd, codigo = "conectada", 200
    except psycopg2.Error:
        bd, codigo = "error", 503
    return jsonify(
        servicio="ms-actualizar-python",
        lenguaje="Python (Flask)",
        operacion="ACTUALIZACION",
        estado="ok" if codigo == 200 else "error",
        base_de_datos=bd,
        documentacion="/api-docs",
    ), codigo


@app.put("/api/curiosidades/<int:curiosidad_id>")
def actualizar(curiosidad_id):
    data = request.get_json(silent=True) or {}
    especie = (data.get("especie") or "").strip().lower() or None
    texto = (data.get("texto") or "").strip() or None
    if especie is None and texto is None:
        return jsonify(error="envia especie y/o texto"), 400
    if especie is not None and especie not in ESPECIES:
        return jsonify(error="especie invalida"), 400
    try:
        fila = ejecutar(
            "UPDATE curiosidades SET especie = COALESCE(%s, especie), texto = COALESCE(%s, texto) "
            "WHERE id = %s RETURNING id, especie, texto",
            (especie, texto, curiosidad_id),
        )
    except psycopg2.Error:
        app.logger.exception("error actualizando")
        return jsonify(error="Error escribiendo en la base de datos"), 500
    if fila is None:
        return jsonify(error="curiosidad no encontrada"), 404
    return jsonify(id=fila[0], especie=fila[1], texto=fila[2])


# ---------- Documentacion Swagger (/api-docs) ----------
@app.get("/openapi.json")
def openapi():
    return jsonify({
        "openapi": "3.0.0",
        "info": {
            "title": "API - Microservicio de ACTUALIZACION (Python)",
            "version": "1.0.0",
            "description": "Actualiza curiosidades de mascotas en la tabla curiosidades de Neon.",
        },
        "servers": [{"url": request.host_url.rstrip("/")}],
        "paths": {
            "/": {"get": {
                "summary": "Estado del servicio",
                "responses": {
                    "200": {"description": "Servicio y base de datos funcionando"},
                    "503": {"description": "Sin conexion con la base de datos"},
                },
            }},
            "/api/curiosidades/{id}": {"put": {
                "summary": "Actualizar una curiosidad",
                "parameters": [{"in": "path", "name": "id", "required": True,
                                "schema": {"type": "integer", "example": 1}}],
                "requestBody": {"required": True, "content": {"application/json": {"schema": {
                    "type": "object",
                    "properties": {
                        "especie": {"type": "string", "enum": ["perro", "gato", "dragon", "robot"], "example": "gato"},
                        "texto": {"type": "string", "example": "Texto actualizado."},
                    },
                }}}},
                "responses": {
                    "200": {"description": "Curiosidad actualizada (devuelve id, especie y texto)"},
                    "400": {"description": "Datos invalidos"},
                    "404": {"description": "Curiosidad no encontrada"},
                    "500": {"description": "Error escribiendo en la base de datos"},
                },
            }},
        },
    })


@app.get("/api-docs")
def api_docs():
    return """<!doctype html>
<html><head><meta charset="utf-8"><title>API - Actualizar (Python)</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({ url: "/openapi.json", dom_id: "#swagger-ui" });</script>
</body></html>"""