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
const PUERTO = process.env.PORT || 4001;

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
        servicio: 'ms-insertar-node',
        lenguaje: 'Node.js (Express)',
        operacion: 'INSERCION',
        estado: bd === 'conectada' ? 'ok' : 'error',
        base_de_datos: bd,
        documentacion: '/api-docs',
    });
});

// ---------- Insertar ----------
app.post('/api/curiosidades', async (req, res) => {
    const body = req.body || {};
    const especie = String(body.especie || '').trim().toLowerCase();
    const texto = String(body.texto || '').trim();
    if (!ESPECIES.includes(especie) || !texto) {
        return res.status(400).json({ error: 'especie valida (perro, gato, dragon, robot) y texto son obligatorios' });
    }
    try {
        const r = await pool.query(
            'INSERT INTO curiosidades (especie, texto) VALUES ($1, $2) RETURNING id, especie, texto',
            [especie, texto]
        );
        res.status(201).json(r.rows[0]);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Error escribiendo en la base de datos' });
    }
});

// ---------- Documentacion Swagger (/api-docs) ----------
const swaggerSpec = {
    openapi: '3.0.0',
    info: {
        title: 'API - Microservicio de INSERCION (Node.js)',
        version: '1.0.0',
        description: 'Inserta curiosidades de mascotas en la tabla curiosidades de Neon.',
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
        '/api/curiosidades': {
            post: {
                summary: 'Insertar una curiosidad',
                requestBody: {
                    required: true,
                    content: {
                        'application/json': {
                            schema: {
                                type: 'object',
                                required: ['especie', 'texto'],
                                properties: {
                                    especie: { type: 'string', enum: ESPECIES, example: 'gato' },
                                    texto: { type: 'string', example: 'Los gatos ronronean al ser felices.' },
                                },
                            },
                        },
                    },
                },
                responses: {
                    201: { description: 'Curiosidad creada (devuelve id, especie y texto)' },
                    400: { description: 'Datos invalidos' },
                    500: { description: 'Error escribiendo en la base de datos' },
                },
            },
        },
    },
};
app.use('/api-docs', swaggerUi.serve, swaggerUi.setup(swaggerSpec));

app.listen(PUERTO, () => {
    console.log(`ms-insertar-node escuchando en el puerto ${PUERTO}`);
});
