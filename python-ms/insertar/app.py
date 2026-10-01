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
    return jsonify(status="ok", servicio="ms-insertar-python")


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
