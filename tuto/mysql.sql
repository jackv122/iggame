

-- Example: create daily partitions for Aug 2025
-- CALL add_betting_partitions_daily('2025-08-01','2025-09-01');

mysql --host=localhost --port=3306 --user=root --password=hailuava12a6 vwallet_001
mysql --host=localhost --port=3306 --user=root --password=hailuava12a6 vgame

-- DB wallet
CREATE TABLE `betting` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `gameid` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `gamenumber` bigint NOT NULL,
  `roomid` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `userid` int NOT NULL,
  `roundid` int NOT NULL,
  `betdetail` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `result` tinytext CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `payout` double NOT NULL DEFAULT '0',
  `payedout` tinyint NOT NULL DEFAULT '0',
  `updatetime` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `h` varchar(44) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `rollback` tinyint DEFAULT '0',
  PRIMARY KEY (`id`,`updatetime`),
  KEY `gameId` (`gameid`),
  KEY `roomId` (`roomid`),
  KEY `userid` (`userid`),
  KEY `gamenumber` (`gamenumber`),
  KEY `idx_user_time` (`userid`,`updatetime`,`id`),
  KEY `idx_game_time` (`gameid`,`updatetime`,`id`)
) ENGINE=InnoDB AUTO_INCREMENT=0 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- DB vgame
CREATE TABLE `gamestate` (
  `gamenumber` bigint NOT NULL AUTO_INCREMENT,
  `gameid` varchar(20) COLLATE utf8mb4_general_ci NOT NULL,
  `roundid` int NOT NULL,
  `state` int NOT NULL,
  `statetime` double NOT NULL,
  `data` text COLLATE utf8mb4_general_ci,
  `result` text COLLATE utf8mb4_general_ci,
  `updatetime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `tx` varchar(100) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `w` varchar(45) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `h` varchar(44) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`gamenumber`)
) ENGINE=InnoDB AUTO_INCREMENT=0 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `trend` (
  `gamenumber` bigint NOT NULL AUTO_INCREMENT,
  `gameid` varchar(20) COLLATE utf8mb4_general_ci NOT NULL,
  `roundid` int NOT NULL,
  `result` varchar(45) COLLATE utf8mb4_general_ci NOT NULL,
  `data` text COLLATE utf8mb4_general_ci,
  `updatetime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `tx` varchar(100) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `w` varchar(45) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `h` varchar(44) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`gamenumber`),
  KEY `createtime` (`updatetime` DESC)
) ENGINE=InnoDB AUTO_INCREMENT=0 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


mysqldump -u root -p --no-data vwallet_001 > vwallet_001.sql
https://dbdiagram.io/d
