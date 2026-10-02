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
        servicio="ms-eliminar-python",
        lenguaje="Python (Flask)",
        operacion="ELIMINACION",
        estado="ok" if codigo == 200 else "error",
        base_de_datos=bd,
        documentacion="/api-docs",
    ), codigo


@app.delete("/api/curiosidades/<int:curiosidad_id>")
def eliminar(curiosidad_id):
    try:
        fila = ejecutar("DELETE FROM curiosidades WHERE id = %s RETURNING id", (curiosidad_id,))
    except psycopg2.Error:
        app.logger.exception("error eliminando")
        return jsonify(error="Error escribiendo en la base de datos"), 500
    if fila is None:
        return jsonify(error="curiosidad no encontrada"), 404
    return jsonify(eliminado=fila[0])


# ---------- Documentacion Swagger (/api-docs) ----------
@app.get("/openapi.json")
def openapi():
    return jsonify({
        "openapi": "3.0.0",
        "info": {
            "title": "API - Microservicio de ELIMINACION (Python)",
            "version": "1.0.0",
            "description": "Elimina curiosidades de mascotas de la tabla curiosidades de Neon.",
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
            "/api/curiosidades/{id}": {"delete": {
                "summary": "Eliminar una curiosidad",
                "parameters": [{"in": "path", "name": "id", "required": True,
                                "schema": {"type": "integer", "example": 1}}],
                "responses": {
                    "200": {"description": "Curiosidad eliminada (devuelve el id eliminado)"},
                    "404": {"description": "Curiosidad no encontrada"},
                    "500": {"description": "Error escribiendo en la base de datos"},
                },
            }},
        },
    })


@app.get("/api-docs")
def api_docs():
    return """<!doctype html>
<html><head><meta charset="utf-8"><title>API - Eliminar (Python)</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({ url: "/openapi.json", dom_id: "#swagger-ui" });</script>
</body></html>"""