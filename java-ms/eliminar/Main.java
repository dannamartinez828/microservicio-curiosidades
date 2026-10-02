import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.URI;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Properties;
import java.util.concurrent.Executors;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public class Main {

    static final String SERVICIO = "ms-eliminar-java";
    static final String OPERACION = "ELIMINACION";
    static final Gson GSON = new GsonBuilder().setPrettyPrinting().disableHtmlEscaping().create();
    static final Pattern RUTA_ID = Pattern.compile("^/api/curiosidades/(\\d+)$");
    static final String ESPECIFICACION = """
            {
              "openapi": "3.0.0",
              "info": {"title": "API - Microservicio de ELIMINACION (Java)", "version": "1.0.0", "description": "Elimina curiosidades de mascotas de la tabla curiosidades de Neon."},
              "servers": [{"url": "%s"}],
              "paths": {
                "/": {"get": {"summary": "Estado del servicio", "responses": {"200": {"description": "Servicio y base de datos funcionando"}, "503": {"description": "Sin conexion con la base de datos"}}}},
                "/api/curiosidades/{id}": {"delete": {"summary": "Eliminar una curiosidad", "parameters": [{"in": "path", "name": "id", "required": true, "schema": {"type": "integer", "example": 1}}], "responses": {"200": {"description": "Curiosidad eliminada (devuelve el id eliminado)"}, "404": {"description": "Curiosidad no encontrada"}, "500": {"description": "Error escribiendo en la base de datos"}}}}
              }
            }
            """;

    static String jdbcUrl;
    static Properties props;

    // ---------- Conexion a Neon (convierte DATABASE_URL a formato JDBC) ----------
    static void configurarBD() throws Exception {
        String raw = System.getenv("DATABASE_URL");
        if (raw == null || raw.isBlank()) {
            throw new IllegalStateException("Falta la variable de entorno DATABASE_URL");
        }
        URI uri = new URI(raw.replaceFirst("^postgres(ql)?://", "postgresql://"));
        String[] usuario = uri.getRawUserInfo().split(":", 2);
        int puerto = uri.getPort() == -1 ? 5432 : uri.getPort();
        String sslmode = "prefer";
        if (uri.getQuery() != null) {
            for (String par : uri.getQuery().split("&")) {
                if (par.startsWith("sslmode=")) sslmode = par.substring("sslmode=".length());
            }
        }
        jdbcUrl = "jdbc:postgresql://" + uri.getHost() + ":" + puerto + uri.getPath();
        props = new Properties();
        props.setProperty("user", URLDecoder.decode(usuario[0], StandardCharsets.UTF_8));
        props.setProperty("password", usuario.length > 1 ? URLDecoder.decode(usuario[1], StandardCharsets.UTF_8) : "");
        props.setProperty("sslmode", sslmode);
        props.setProperty("connectTimeout", "10");
        props.setProperty("socketTimeout", "30");
    }

    static Connection conectar() throws SQLException {
        return DriverManager.getConnection(jdbcUrl, props);
    }

    public static void main(String[] args) throws Exception {
        configurarBD();
        int puerto = Integer.parseInt(System.getenv().getOrDefault("PORT", "6003"));
        HttpServer servidor = HttpServer.create(new InetSocketAddress(puerto), 0);
        servidor.createContext("/", Main::manejar);
        servidor.setExecutor(Executors.newFixedThreadPool(8));
        servidor.start();
        System.out.println(SERVICIO + " escuchando en el puerto " + puerto);
    }

    // ---------- Enrutador ----------
    static void manejar(HttpExchange ex) throws IOException {
        try {
            String ruta = ex.getRequestURI().getPath();
            String metodo = ex.getRequestMethod();

            if (ruta.equals("/")) {
                if (metodo.equals("GET")) estado(ex); else noPermitido(ex);
            } else if (ruta.equals("/openapi.json")) {
                if (metodo.equals("GET")) openapi(ex); else noPermitido(ex);
            } else if (ruta.equals("/api-docs")) {
                if (metodo.equals("GET")) apiDocs(ex); else noPermitido(ex);
            } else if (RUTA_ID.matcher(ruta).matches()) {
                if (metodo.equals("DELETE")) eliminar(ex, idDe(ruta)); else noPermitido(ex);
            } else {
                enviar(ex, 404, error("ruta no encontrada"));
            }
        } catch (Exception e) {
            e.printStackTrace();
            enviar(ex, 500, error("Error interno del servidor"));
        } finally {
            ex.close();
        }
    }

    // ---------- Estado del servicio (raiz) ----------
    static void estado(HttpExchange ex) throws IOException {
        boolean ok;
        try (Connection c = conectar(); Statement st = c.createStatement()) {
            st.execute("SELECT 1");
            ok = true;
        } catch (SQLException e) {
            ok = false;
        }
        Map<String, Object> r = new LinkedHashMap<>();
        r.put("servicio", SERVICIO);
        r.put("lenguaje", "Java");
        r.put("operacion", OPERACION);
        r.put("estado", ok ? "ok" : "error");
        r.put("base_de_datos", ok ? "conectada" : "error");
        r.put("documentacion", "/api-docs");
        enviar(ex, ok ? 200 : 503, r);
    }

    // ---------- Eliminar ----------
    static long idDe(String ruta) {
        Matcher m = RUTA_ID.matcher(ruta);
        m.matches();
        try {
            return Long.parseLong(m.group(1));
        } catch (NumberFormatException e) {
            return -1;
        }
    }

    static void eliminar(HttpExchange ex, long id) throws IOException {
        try (Connection c = conectar();
             PreparedStatement ps = c.prepareStatement("DELETE FROM curiosidades WHERE id = ? RETURNING id")) {
            ps.setLong(1, id);
            try (ResultSet rs = ps.executeQuery()) {
                if (!rs.next()) {
                    enviar(ex, 404, error("curiosidad no encontrada"));
                    return;
                }
                Map<String, Object> r = new LinkedHashMap<>();
                r.put("eliminado", rs.getInt("id"));
                enviar(ex, 200, r);
            }
        } catch (SQLException e) {
            e.printStackTrace();
            enviar(ex, 500, error("Error escribiendo en la base de datos"));
        }
    }

    // ---------- Documentacion Swagger (/api-docs) ----------
    static void openapi(HttpExchange ex) throws IOException {
        enviarTexto(ex, 200, "application/json; charset=utf-8", ESPECIFICACION.formatted(servidorPublico(ex)));
    }

    static void apiDocs(HttpExchange ex) throws IOException {
        String html = """
                <!doctype html>
                <html><head><meta charset="utf-8"><title>API - Eliminar (Java)</title>
                <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"></head>
                <body><div id="swagger-ui"></div>
                <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
                <script>SwaggerUIBundle({ url: "/openapi.json", dom_id: "#swagger-ui" });</script>
                </body></html>
                """;
        enviarTexto(ex, 200, "text/html; charset=utf-8", html);
    }

    // URL publica del servicio (respeta https detras del proxy de Render)
    static String servidorPublico(HttpExchange ex) {
        String proto = ex.getRequestHeaders().getFirst("X-Forwarded-Proto");
        if (proto == null || proto.isBlank()) proto = "http";
        proto = proto.split(",")[0].trim();
        String host = ex.getRequestHeaders().getFirst("Host");
        return proto + "://" + (host == null ? "localhost" : host);
    }

    // ---------- Utilidades ----------
    static JsonObject leerJson(HttpExchange ex) {
        try {
            String texto = new String(ex.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
            JsonElement e = JsonParser.parseString(texto);
            return e.isJsonObject() ? e.getAsJsonObject() : new JsonObject();
        } catch (Exception e) {
            return new JsonObject();
        }
    }

    static String campo(JsonObject o, String nombre) {
        JsonElement e = o.get(nombre);
        if (e == null || !e.isJsonPrimitive()) return "";
        return e.getAsString().strip();
    }

    static Map<String, Object> error(String mensaje) {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("error", mensaje);
        return m;
    }

    static void noPermitido(HttpExchange ex) throws IOException {
        enviar(ex, 405, error("metodo no permitido"));
    }

    static void enviar(HttpExchange ex, int codigo, Object cuerpo) throws IOException {
        enviarTexto(ex, codigo, "application/json; charset=utf-8", GSON.toJson(cuerpo));
    }

    static void enviarTexto(HttpExchange ex, int codigo, String tipo, String texto) throws IOException {
        byte[] bytes = texto.getBytes(StandardCharsets.UTF_8);
        ex.getResponseHeaders().set("Content-Type", tipo);
        ex.sendResponseHeaders(codigo, bytes.length);
        ex.getResponseBody().write(bytes);
    }
}
