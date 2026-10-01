import os
from flask import Flask,jsonify
import psycopg
app=Flask(__name__)
def conn(): return psycopg.connect(os.environ["DATABASE_URL"],sslmode="require")
@app.get("/health")
def health(): return jsonify(status="ok",servicio="consulta-python")
@app.get("/api/mascotas")
def mascotas():
 try:
  with conn() as c:
   with c.cursor() as cur:
    cur.execute("SELECT id,nombre,especie,hambre,felicidad,fecha_creacion,ultima_interaccion FROM mascotas_mascota ORDER BY id")
    cols=[d.name for d in cur.description]
    return jsonify([dict(zip(cols,r)) for r in cur.fetchall()])
 except Exception as e: print(e); return jsonify(error="Error consultando mascotas"),500
@app.get("/api/curiosidades/<especie>")
def curiosidades(especie):
 try:
  with conn() as c:
   with c.cursor() as cur:
    cur.execute("SELECT id,especie,texto FROM curiosidades WHERE especie=%s",[especie])
    cols=[d.name for d in cur.description]
    return jsonify([dict(zip(cols,r)) for r in cur.fetchall()])
 except Exception as e: print(e); return jsonify(error="Error consultando curiosidades"),500
app.run(host="0.0.0.0",port=int(os.getenv("PORT","10000")))
