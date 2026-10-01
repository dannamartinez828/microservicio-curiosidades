import os
from flask import Flask,request,jsonify
import psycopg
app=Flask(__name__)
def conn(): return psycopg.connect(os.environ["DATABASE_URL"],sslmode="require")
def datos():
 b=request.get_json(silent=True) or {}
 return {"nombre":str(b.get("nombre","")).strip(),"especie":str(b.get("especie","perro")).strip(),"hambre":int(b.get("hambre",50)),"felicidad":int(b.get("felicidad",50))}
def rd(cur,row): return dict(zip([d.name for d in cur.description],row))
@app.get("/health")
def health(): return jsonify(status="ok",servicio=os.getenv("SERVICE_NAME","python"))
@app.post("/api/mascotas")
def insertar():
 d=datos()
 if not d["nombre"]: return jsonify(error="nombre es obligatorio"),400
 try:
  with conn() as c:
   with c.cursor() as cur:
    cur.execute("INSERT INTO mascotas_mascota (nombre,especie,hambre,felicidad,fecha_creacion,ultima_interaccion) VALUES (%s,%s,%s,%s,NOW(),NOW()) RETURNING *",(d["nombre"],d["especie"],d["hambre"],d["felicidad"]))
    return jsonify(rd(cur,cur.fetchone())),201
 except Exception as e: print(e); return jsonify(error="Error insertando mascota"),500
app.run(host="0.0.0.0",port=int(os.getenv("PORT","10000")))
