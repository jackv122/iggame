

-- Example: create daily partitions for Aug 2025
-- CALL add_betting_partitions_daily('2025-08-01','2025-09-01');

mysql --host=localhost --port=3306 --user=root --password=hailuava12a6 vwallet_001

ALTER TABLE betting
  MODIFY COLUMN updatetime DATETIME NOT NULL
  DEFAULT CURRENT_TIMESTAMP
  ON UPDATE CURRENT_TIMESTAMP;

-- Adjust PK to include the partition key
ALTER TABLE betting
  DROP PRIMARY KEY,
  ADD PRIMARY KEY (id, updatetime),
  DROP INDEX createtime,
  ADD INDEX idx_user_time (userid, updatetime, id),
  ADD INDEX idx_game_time (gameid, updatetime, id);

-- DATABASE: vwallet_001
INSERT INTO betting
  (gameid, gamenumber, roomid, userid, roundid, betdetail, result, payout, payedout, h)
VALUES
  ('001', 123456789, '000001', 42, 1, '{"bets":[{"type":"F7","amt":1.5}]}', '7', 0, 0, 'txhash_or_sig_here');

CREATE TABLE `betting` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `gameid` varchar(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `gamenumber` bigint NOT NULL,
  `roomid` varchar(6) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
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
) ENGINE=InnoDB AUTO_INCREMENT=3136 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci


--EX:
    ALTER TABLE betting PARTITION BY RANGE COLUMNS(updatetime) (
        PARTITION p2025_01 VALUES LESS THAN ('2025-02-01'),
        PARTITION p2025_02 VALUES LESS THAN ('2025-03-01'),
        PARTITION p2025_03 VALUES LESS THAN ('2025-04-01'),
        PARTITION pmax VALUES LESS THAN (MAXVALUE)
    );



-- Add next month --------------------------------
SET @first_day_next := DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 1 MONTH), '%Y-%m-01');
SET @pname := DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 0 MONTH), 'p%Y_%m');
SET @sql := CONCAT(
  'ALTER TABLE betting ADD PARTITION (PARTITION ', @pname,
  ' VALUES LESS THAN (''', @first_day_next, '''))'
);
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;

-- EX: 
    ALTER TABLE betting
    ADD PARTITION (
    PARTITION p2025_10 VALUES LESS THAN ('2025-11-01')
    );

-- drop old month --------------------------------
SET @old := DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 12 MONTH), 'p%Y_%m');
SET @sql := CONCAT('ALTER TABLE betting DROP PARTITION ', @old);
-- You may want to check information_schema.PARTITIONS first
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;
-- EX:
    ALTER TABLE betting DROP PARTITION p2025_01;

    ALTER TABLE betting DROP PARTITION (
        p2025_01,
        p2025_02
    );



-- DATABASE: vgame
CREATE TABLE `trend` (
  `gamenumber` bigint NOT NULL AUTO_INCREMENT,
  `gameid` varchar(20) COLLATE utf8mb4_general_ci NOT NULL,
  `roundid` int NOT NULL,
  `result` varchar(45) COLLATE utf8mb4_general_ci NOT NULL,
  `data` tinytext COLLATE utf8mb4_general_ci,
  `updatetime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `tx` varchar(100) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `w` varchar(45) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  `h` varchar(44) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`gamenumber`),
  KEY `createtime` (`updatetime` DESC)
) ENGINE=InnoDB AUTO_INCREMENT=10771 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
