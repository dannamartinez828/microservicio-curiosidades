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
@app.delete("/api/mascotas/<int:id>")
def eliminar(id):
 try:
  with conn() as c:
   with c.cursor() as cur:
    cur.execute("DELETE FROM mascotas_mascota WHERE id=%s RETURNING id",(id,)); x=cur.fetchone()
    if not x:return jsonify(error="Mascota no encontrada"),404
    return jsonify(ok=True,id=x[0])
 except Exception as e: print(e); return jsonify(error="Error eliminando mascota"),500
app.run(host="0.0.0.0",port=int(os.getenv("PORT","10000")))
