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
@app.put("/api/mascotas/<int:id>")
def actualizar(id):
 d=datos()
 if not d["nombre"]: return jsonify(error="nombre es obligatorio"),400
 try:
  with conn() as c:
   with c.cursor() as cur:
    cur.execute("UPDATE mascotas_mascota SET nombre=%s,especie=%s,hambre=%s,felicidad=%s,ultima_interaccion=NOW() WHERE id=%s RETURNING *",(d["nombre"],d["especie"],d["hambre"],d["felicidad"],id))
    x=cur.fetchone()
    if not x:return jsonify(error="Mascota no encontrada"),404
    return jsonify(rd(cur,x))
 except Exception as e: print(e); return jsonify(error="Error actualizando mascota"),500
app.run(host="0.0.0.0",port=int(os.getenv("PORT","10000")))
