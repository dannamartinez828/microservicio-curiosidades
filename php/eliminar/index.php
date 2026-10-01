<?php
header('Content-Type: application/json; charset=utf-8'); header('Access-Control-Allow-Origin: *');
if ($_SERVER['REQUEST_METHOD']==='OPTIONS'){http_response_code(204);exit;}
function out($x,$c=200){http_response_code($c);echo json_encode($x,JSON_UNESCAPED_UNICODE);exit;}
function db(){ $u=parse_url(getenv('DATABASE_URL')); if(!$u) throw new Exception('DATABASE_URL no configurada'); $dsn='pgsql:host='.$u['host'].';port='.($u['port']??5432).';dbname='.ltrim($u['path'],'/'); return new PDO($dsn,$u['user'],$u['pass'],[PDO::ATTR_ERRMODE=>PDO::ERRMODE_EXCEPTION,PDO::ATTR_DEFAULT_FETCH_MODE=>PDO::FETCH_ASSOC,PDO::PGSQL_ATTR_SSL_MODE=>'require']);}
function body(){ $b=json_decode(file_get_contents('php://input'),true)?:[]; return ['nombre'=>trim((string)($b['nombre']??'')),'especie'=>trim((string)($b['especie']??'perro')),'hambre'=>(int)($b['hambre']??50),'felicidad'=>(int)($b['felicidad']??50)];}
if ($_SERVER['REQUEST_URI']==='/health') out(['status'=>'ok','servicio'=>getenv('SERVICE_NAME')?:'php']);
try{$id=(int)basename(parse_url($_SERVER['REQUEST_URI'],PHP_URL_PATH));$p=db();$s=$p->prepare('DELETE FROM mascotas_mascota WHERE id=? RETURNING id');$s->execute([$id]);$r=$s->fetch();if(!$r)out(['error'=>'Mascota no encontrada'],404);out(['ok'=>true,'id'=>(int)$r['id']]);}catch(Throwable $e){error_log($e->getMessage());out(['error'=>'Error eliminando mascota'],500);}