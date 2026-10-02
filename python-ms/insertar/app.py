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
        servicio="ms-insertar-python",
        lenguaje="Python (Flask)",
        operacion="INSERCION",
        estado="ok" if codigo == 200 else "error",
        base_de_datos=bd,
        documentacion="/api-docs",
    ), codigo


@app.post("/api/curiosidades")
def insertar():
    data = request.get_json(silent=True) or {}
    especie = (data.get("especie") or "").strip().lower()
    texto = (data.get("texto") or "").strip()
    if especie not in ESPECIES or not texto:
        return jsonify(error="especie valida (perro, gato, dragon, robot) y texto son obligatorios"), 400
    try:
        fila = ejecutar(
            "INSERT INTO curiosidades (especie, texto) VALUES (%s, %s) RETURNING id, especie, texto",
            (especie, texto),
        )
    except psycopg2.Error:
        app.logger.exception("error insertando")
        return jsonify(error="Error escribiendo en la base de datos"), 500
    return jsonify(id=fila[0], especie=fila[1], texto=fila[2]), 201


# ---------- Documentacion Swagger (/api-docs) ----------
@app.get("/openapi.json")
def openapi():
    return jsonify({
        "openapi": "3.0.0",
        "info": {
            "title": "API - Microservicio de INSERCION (Python)",
            "version": "1.0.0",
            "description": "Inserta curiosidades de mascotas en la tabla curiosidades de Neon.",
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
            "/api/curiosidades": {"post": {
                "summary": "Insertar una curiosidad",
                "requestBody": {"required": True, "content": {"application/json": {"schema": {
                    "type": "object",
                    "required": ["especie", "texto"],
                    "properties": {
                        "especie": {"type": "string", "enum": ["perro", "gato", "dragon", "robot"], "example": "gato"},
                        "texto": {"type": "string", "example": "Los gatos ronronean al ser felices."},
                    },
                }}}},
                "responses": {
                    "201": {"description": "Curiosidad creada (devuelve id, especie y texto)"},
                    "400": {"description": "Datos invalidos"},
                    "500": {"description": "Error escribiendo en la base de datos"},
                },
            }},
        },
    })


@app.get("/api-docs")
def api_docs():
    return """<!doctype html>
<html><head><meta charset="utf-8"><title>API - Insertar (Python)</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({ url: "/openapi.json", dom_id: "#swagger-ui" });</script>
</body></html>"""