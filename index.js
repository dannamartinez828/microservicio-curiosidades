require('dotenv').config();
const express = require('express');
const cors = require('cors');
const { Pool } = require('pg');

const app = express();
app.use(cors());

// conexion a la base de datos Neon (PostgreSQL en la nube)
const pool = new Pool({
    connectionString: process.env.DATABASE_URL,
    ssl: { rejectUnauthorized: false }, // Neon exige SSL
});

// ruta de salud, util para confirmar que el servicio esta arriba en Render
app.get('/', (req, res) => {
    res.json({ status: 'ok', mensaje: 'microservicio-curiosidades funcionando' });
});

// ruta dinamica: /api/curiosidades/:especie -> curiosidades de esa especie
app.get('/api/curiosidades/:especie', async (req, res) => {
    const { especie } = req.params;
    try {
        const resultado = await pool.query(
            'SELECT id, especie, texto FROM curiosidades WHERE especie = $1',
            [especie]
        );
        res.json(resultado.rows);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error consultando la base de datos' });
    }
});

// ruta extra: todas las curiosidades, sin filtrar
app.get('/api/curiosidades', async (req, res) => {
    try {
        const resultado = await pool.query('SELECT id, especie, texto FROM curiosidades');
        res.json(resultado.rows);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error consultando la base de datos' });
    }
});

const PUERTO = process.env.PORT || 3000;
app.listen(PUERTO, () => {
    console.log(`microservicio-curiosidades escuchando en el puerto ${PUERTO}`);
});
