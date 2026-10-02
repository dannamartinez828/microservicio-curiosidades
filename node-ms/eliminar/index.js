const express = require('express');
const { Pool } = require('pg');
const swaggerUi = require('swagger-ui-express');

const app = express();
app.use(express.json());
app.set('json spaces', 2);

const pool = new Pool({
    connectionString: process.env.DATABASE_URL,
    ssl: { rejectUnauthorized: false }, // Neon exige SSL
});

const PUERTO = process.env.PORT || 4003;

// ---------- Estado del servicio (raiz) ----------
app.get('/', async (req, res) => {
    let bd = 'conectada';
    try {
        await Promise.race([
            pool.query('SELECT 1'),
            new Promise((_, rechazar) => setTimeout(() => rechazar(new Error('timeout')), 10000)),
        ]);
    } catch (err) {
        bd = 'error';
    }
    res.status(bd === 'conectada' ? 200 : 503).json({
        servicio: 'ms-eliminar-node',
        lenguaje: 'Node.js (Express)',
        operacion: 'ELIMINACION',
        estado: bd === 'conectada' ? 'ok' : 'error',
        base_de_datos: bd,
        documentacion: '/api-docs',
    });
});

// ---------- Eliminar ----------
app.delete('/api/curiosidades/:id(\\d+)', async (req, res) => {
    try {
        const r = await pool.query(
            'DELETE FROM curiosidades WHERE id = $1 RETURNING id',
            [Number(req.params.id)]
        );
        if (r.rows.length === 0) {
            return res.status(404).json({ error: 'curiosidad no encontrada' });
        }
        res.json({ eliminado: r.rows[0].id });
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error escribiendo en la base de datos' });
    }
});

// ---------- Documentacion Swagger (/api-docs) ----------
const swaggerSpec = {
    openapi: '3.0.0',
    info: {
        title: 'API - Microservicio de ELIMINACION (Node.js)',
        version: '1.0.0',
        description: 'Elimina curiosidades de mascotas de la tabla curiosidades de Neon.',
    },
    servers: [{ url: process.env.RENDER_EXTERNAL_URL || `http://localhost:${PUERTO}` }],
    paths: {
        '/': {
            get: {
                summary: 'Estado del servicio',
                responses: {
                    200: { description: 'Servicio y base de datos funcionando' },
                    503: { description: 'Sin conexion con la base de datos' },
                },
            },
        },
        '/api/curiosidades/{id}': {
            delete: {
                summary: 'Eliminar una curiosidad',
                parameters: [{ in: 'path', name: 'id', required: true, schema: { type: 'integer', example: 1 } }],
                responses: {
                    200: { description: 'Curiosidad eliminada (devuelve el id eliminado)' },
                    404: { description: 'Curiosidad no encontrada' },
                    500: { description: 'Error escribiendo en la base de datos' },
                },
            },
        },
    },
};
app.use('/api-docs', swaggerUi.serve, swaggerUi.setup(swaggerSpec));

app.listen(PUERTO, () => {
    console.log(`ms-eliminar-node escuchando en el puerto ${PUERTO}`);
});
