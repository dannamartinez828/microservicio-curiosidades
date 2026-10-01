import os
from contextlib import closing

import psycopg2
from flask import Flask, jsonify

app = Flask(__name__)
DATABASE_URL = os.environ["DATABASE_URL"]


def ejecutar(sql, params):
    with closing(psycopg2.connect(DATABASE_URL)) as conn:
        with conn, conn.cursor() as cur:
            cur.execute(sql, params)
            return cur.fetchone()


@app.get("/health")
def health():
    return jsonify(status="ok", servicio="ms-eliminar-python")


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
