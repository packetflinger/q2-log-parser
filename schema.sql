CREATE TABLE IF NOT EXISTS "rename" (
	"id"	INTEGER,
	"timestamp"	INTEGER,
	"server"	INTEGER,
	"name1"	TEXT,
	"name2"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "connect" (
	"id"	INTEGER,
	"timestamp"	INTEGER,
	"server"	INTEGER,
	"name"	TEXT,
	"ip"	TEXT,
	"client"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "server" (
	"id"	INTEGER,
	"servername"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "chat" (
	"id"	INTEGER,
	"timestamp"	INTEGER,
	"server"	INTEGER,
	"name"	TEXT,
	"team"	INTEGER,
	"msg"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "rcon" (
	"id"	INTEGER,
	"timestamp"	INTEGER,
	"server"	INTEGER,
	"ip"	TEXT,
	"limited"	INTEGER,
	"invalid"	INTEGER,
	"command"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "chat_private" (
	"id"	INTEGER,
	"timestamp"	INTEGER,
	"server"	INTEGER,
	"name1"	TEXT,
	"name2"	TEXT,
	"msg"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
