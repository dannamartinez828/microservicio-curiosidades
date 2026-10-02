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

const ESPECIES = ['perro', 'gato', 'dragon', 'robot'];
const PUERTO = process.env.PORT || 4002;

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
        servicio: 'ms-actualizar-node',
        lenguaje: 'Node.js (Express)',
        operacion: 'ACTUALIZACION',
        estado: bd === 'conectada' ? 'ok' : 'error',
        base_de_datos: bd,
        documentacion: '/api-docs',
    });
});

// ---------- Actualizar ----------
app.put('/api/curiosidades/:id(\\d+)', async (req, res) => {
    const body = req.body || {};
    const especie = String(body.especie || '').trim().toLowerCase() || null;
    const texto = String(body.texto || '').trim() || null;
    if (especie === null && texto === null) {
        return res.status(400).json({ error: 'envia especie y/o texto' });
    }
    if (especie !== null && !ESPECIES.includes(especie)) {
        return res.status(400).json({ error: 'especie invalida' });
    }
    try {
        const r = await pool.query(
            'UPDATE curiosidades SET especie = COALESCE($1, especie), texto = COALESCE($2, texto) ' +
            'WHERE id = $3 RETURNING id, especie, texto',
            [especie, texto, Number(req.params.id)]
        );
        if (r.rows.length === 0) {
            return res.status(404).json({ error: 'curiosidad no encontrada' });
        }
        res.json(r.rows[0]);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error escribiendo en la base de datos' });
    }
});

// ---------- Documentacion Swagger (/api-docs) ----------
const swaggerSpec = {
    openapi: '3.0.0',
    info: {
        title: 'API - Microservicio de ACTUALIZACION (Node.js)',
        version: '1.0.0',
        description: 'Actualiza curiosidades de mascotas en la tabla curiosidades de Neon.',
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
            put: {
                summary: 'Actualizar una curiosidad',
                parameters: [{ in: 'path', name: 'id', required: true, schema: { type: 'integer', example: 1 } }],
                requestBody: {
                    required: true,
                    content: {
                        'application/json': {
                            schema: {
                                type: 'object',
                                properties: {
                                    especie: { type: 'string', enum: ESPECIES, example: 'gato' },
                                    texto: { type: 'string', example: 'Texto actualizado.' },
                                },
                            },
                        },
                    },
                },
                responses: {
                    200: { description: 'Curiosidad actualizada (devuelve id, especie y texto)' },
                    400: { description: 'Datos invalidos' },
                    404: { description: 'Curiosidad no encontrada' },
                    500: { description: 'Error escribiendo en la base de datos' },
                },
            },
        },
    },
};
app.use('/api-docs', swaggerUi.serve, swaggerUi.setup(swaggerSpec));

app.listen(PUERTO, () => {
    console.log(`ms-actualizar-node escuchando en el puerto ${PUERTO}`);
});
