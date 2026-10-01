package com.mascotas;
import com.sun.net.httpserver.*;
import java.io.*; import java.net.*; import java.nio.charset.StandardCharsets; import java.sql.*;
import java.util.*;
public class App {
 static String db(){return System.getenv("DATABASE_URL");}
 static Connection conn() throws Exception { String u=db(); if(u==null)throw new Exception("DATABASE_URL no configurada"); if(u.startsWith("postgresql://")) u="jdbc:"+u; return DriverManager.getConnection(u); }
 static void send(HttpExchange e,int code,String body)throws IOException{e.getResponseHeaders().set("Content-Type","application/json; charset=utf-8");byte[] b=body.getBytes(StandardCharsets.UTF_8);e.sendResponseHeaders(code,b.length);try(OutputStream o=e.getResponseBody()){o.write(b);}}
 static String body(HttpExchange e)throws IOException{return new String(e.getRequestBody().readAllBytes(),StandardCharsets.UTF_8);}
 static String val(String j,String k,String def){String p="\"" + k + "\"\\s*:\\s*\"([^\"]*)\"";var m=java.util.regex.Pattern.compile(p).matcher(j);return m.find()?m.group(1):def;}
 static int num(String j,String k,int def){String p="\"" + k + "\"\\s*:\\s*(-?\\d+)";var m=java.util.regex.Pattern.compile(p).matcher(j);return m.find()?Integer.parseInt(m.group(1)):def;}
 static String json(ResultSet r)throws Exception{return "{\"id\":"+r.getLong("id")+",\"nombre\":"+q(r.getString("nombre"))+",\"especie\":"+q(r.getString("especie"))+",\"hambre\":"+r.getInt("hambre")+",\"felicidad\":"+r.getInt("felicidad")+"}";}
 static String q(String s){return "\""+s.replace("\\","\\\\").replace("\"","\\\"")+"\"";}
 static void crud(HttpExchange e,String op)throws Exception{
   String path=e.getRequestURI().getPath(); String[] a=path.split("/"); int id=a.length>3?Integer.parseInt(a[3]):-1;
   try(Connection c=conn()){
    if(op.equals("insertar")){
      String j=body(e),n=val(j,"nombre",""),es=val(j,"especie","perro");int h=num(j,"hambre",50),f=num(j,"felicidad",50);
      if(n.isBlank()){send(e,400,"{\"error\":\"nombre es obligatorio\"}");return;}
      try(PreparedStatement s=c.prepareStatement("INSERT INTO mascotas_mascota (nombre,especie,hambre,felicidad,fecha_creacion,ultima_interaccion) VALUES (?,?,?,?,NOW(),NOW()) RETURNING *")){s.setString(1,n);s.setString(2,es);s.setInt(3,h);s.setInt(4,f);ResultSet r=s.executeQuery();r.next();send(e,201,json(r));}
    } else if(op.equals("actualizar")){
      String j=body(e),n=val(j,"nombre",""),es=val(j,"especie","perro");int h=num(j,"hambre",50),f=num(j,"felicidad",50);
      try(PreparedStatement s=c.prepareStatement("UPDATE mascotas_mascota SET nombre=?,especie=?,hambre=?,felicidad=?,ultima_interaccion=NOW() WHERE id=? RETURNING *")){s.setString(1,n);s.setString(2,es);s.setInt(3,h);s.setInt(4,f);s.setInt(5,id);ResultSet r=s.executeQuery();if(!r.next()){send(e,404,"{\"error\":\"Mascota no encontrada\"}");return;}send(e,200,json(r));}
    } else {
      try(PreparedStatement s=c.prepareStatement("DELETE FROM mascotas_mascota WHERE id=? RETURNING id")){s.setInt(1,id);ResultSet r=s.executeQuery();if(!r.next()){send(e,404,"{\"error\":\"Mascota no encontrada\"}");return;}send(e,200,"{\"ok\":true,\"id\":"+r.getLong(1)+"}");}
    }
   }catch(Exception x){x.printStackTrace();send(e,500,"{\"error\":\"Error en "+op+" mascota\"}");}
 }
 public static void main(String[] args)throws Exception{int port=Integer.parseInt(System.getenv().getOrDefault("PORT","10000"));HttpServer s=HttpServer.create(new InetSocketAddress(port),0);s.createContext("/health",e->send(e,200,"{\"status\":\"ok\",\"servicio\":\"java\"}"));String op=System.getenv().getOrDefault("OPERATION","insertar");s.createContext("/api/mascotas",e->{try{crud(e,op);}catch(Exception x){send(e,500,"{\"error\":\"Error interno\"}");}});s.start();}
}
