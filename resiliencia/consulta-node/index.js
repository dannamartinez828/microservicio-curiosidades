const express=require('express');const cors=require('cors');const {Pool}=require('pg');
const app=express();app.use(cors());const pool=new Pool({connectionString:process.env.DATABASE_URL,ssl:{rejectUnauthorized:false}});
app.get('/health',(_q,r)=>r.json({status:'ok',servicio:'consulta-node'}));
app.get('/api/mascotas',async(_q,r)=>{try{const x=await pool.query('SELECT id,nombre,especie,hambre,felicidad,fecha_creacion,ultima_interaccion FROM mascotas_mascota ORDER BY id');r.json(x.rows)}catch(e){console.error(e);r.status(500).json({error:'Error consultando mascotas'})}});
app.get('/api/curiosidades/:especie',async(q,r)=>{try{const x=await pool.query('SELECT id,especie,texto FROM curiosidades WHERE especie=$1',[q.params.especie]);r.json(x.rows)}catch(e){console.error(e);r.status(500).json({error:'Error consultando curiosidades'})}});
app.listen(process.env.PORT||10000);
