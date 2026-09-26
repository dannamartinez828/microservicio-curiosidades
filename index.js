require('dotenv').config();
const express = require('express');
const cors = require('cors');
const { Pool } = require('pg');
const swaggerJsdoc = require('swagger-jsdoc');
const swaggerUi = require('swagger-ui-express');

const app = express();
app.use(cors());

// conexion a la base de datos Neon (PostgreSQL en la nube)
const pool = new Pool({
    connectionString: process.env.DATABASE_URL,
    ssl: { rejectUnauthorized: false }, // Neon exige SSL
});

// ---------- Configuracion de Swagger ----------
const swaggerSpec = swaggerJsdoc({
    definition: {
        openapi: '3.0.0',
        info: {
            title: 'API - Microservicio de Curiosidades',
            version: '1.0.0',
            description:
                'Microservicio en Node.js/Express que consulta curiosidades de mascotas ' +
                'guardadas en una base de datos PostgreSQL en Neon. Lo consume la vista ' +
                '"ver_curiosidades" de la app Django de mascotas virtuales.',
        },
        servers: [
            { url: 'http://localhost:3000', description: 'Servidor local' },
            { url: 'https://microservicio-curiosidades.onrender.com', description: 'Render (produccion)' },
        ],
        components: {
            schemas: {
                Curiosidad: {
                    type: 'object',
                    properties: {
                        id: { type: 'integer', example: 1 },
                        especie: { type: 'string', example: 'perro' },
                        texto: { type: 'string', example: 'Los perros pueden aprender mas de 100 palabras.' },
                    },
                },
                Error: {
                    type: 'object',
                    properties: {
                        error: { type: 'string', example: 'Error consultando la base de datos' },
                    },
                },
            },
        },
    },
    apis: ['./index.js'], // aqui mismo estan los comentarios @swagger de cada ruta
});

app.use('/api-docs', swaggerUi.serve, swaggerUi.setup(swaggerSpec));

/**
 * @swagger
 * /:
 *   get:
 *     summary: Health check
 *     description: Confirma que el microservicio esta corriendo (util para verificar el deploy en Render).
 *     responses:
 *       200:
 *         description: El servicio esta arriba
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 status:
 *                   type: string
 *                   example: ok
 *                 mensaje:
 *                   type: string
 *                   example: microservicio-curiosidades funcionando
 */
app.get('/', (req, res) => {
    res.json({ status: 'ok', mensaje: 'microservicio-curiosidades funcionando' });
});

/**
 * @swagger
 * /api/curiosidades/{especie}:
 *   get:
 *     summary: Curiosidades filtradas por especie
 *     description: Ruta dinamica que recibe la especie como parametro y devuelve solo esas curiosidades.
 *     parameters:
 *       - in: path
 *         name: especie
 *         required: true
 *         schema:
 *           type: string
 *           example: perro
 *         description: Especie a filtrar (perro, gato, dragon, robot)
 *     responses:
 *       200:
 *         description: Lista de curiosidades de esa especie
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/Curiosidad'
 *       500:
 *         description: Error consultando la base de datos
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */
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

/**
 * @swagger
 * /api/curiosidades:
 *   get:
 *     summary: Todas las curiosidades
 *     description: Devuelve todas las curiosidades guardadas, sin filtrar por especie.
 *     responses:
 *       200:
 *         description: Lista completa de curiosidades
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/Curiosidad'
 *       500:
 *         description: Error consultando la base de datos
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Error'
 */
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
    console.log(`Documentacion Swagger en http://localhost:${PUERTO}/api-docs`);
});
