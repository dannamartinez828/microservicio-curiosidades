import os
from contextlib import closing

import psycopg2
from flask import Flask, jsonify, request

app = Flask(__name__)
DATABASE_URL = os.environ["DATABASE_URL"]
ESPECIES = {"perro", "gato", "dragon", "robot"}


def ejecutar(sql, params):
    with closing(psycopg2.connect(DATABASE_URL)) as conn:
        with conn, conn.cursor() as cur:
            cur.execute(sql, params)
            return cur.fetchone()


@app.get("/health")
def health():
    return jsonify(status="ok", servicio="ms-actualizar-python")


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
